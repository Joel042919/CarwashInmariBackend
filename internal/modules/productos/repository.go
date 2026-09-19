package productos

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
)

type Repository interface {
	ListarCategorias(ctx context.Context, soloActivas bool) ([]Categoria, error)
	ExisteCategoria(ctx context.Context, id uuid.UUID) (bool, error)
	CrearCategoria(ctx context.Context, c *Categoria) error
	ActualizarCategoria(ctx context.Context, c *Categoria) (bool, error)

	ListarProductos(ctx context.Context, f FiltroProductos) ([]Producto, error)
	ObtenerProducto(ctx context.Context, id uuid.UUID) (*Producto, error)
	CrearProducto(ctx context.Context, p *Producto) error
	ActualizarProducto(ctx context.Context, p *Producto) (bool, error)
	AjustarStock(ctx context.Context, id uuid.UUID, delta int) (ok bool, err error)
}

type repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) Repository {
	return &repository{db: db}
}

func (r *repository) ListarCategorias(ctx context.Context, soloActivas bool) ([]Categoria, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id_categoria, nombre, descripcion, activo
		FROM categorias_producto
		WHERE ($1::boolean = false OR activo)
		ORDER BY nombre`, soloActivas)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	lista := []Categoria{}
	for rows.Next() {
		var c Categoria
		if err := rows.Scan(&c.IDCategoria, &c.Nombre, &c.Descripcion, &c.Activo); err != nil {
			return nil, err
		}
		lista = append(lista, c)
	}
	return lista, rows.Err()
}

func (r *repository) ExisteCategoria(ctx context.Context, id uuid.UUID) (bool, error) {
	var existe bool
	err := r.db.QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM categorias_producto WHERE id_categoria = $1)`, id).Scan(&existe)
	return existe, err
}

func (r *repository) CrearCategoria(ctx context.Context, c *Categoria) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO categorias_producto (id_categoria, nombre, descripcion, activo)
		VALUES ($1, $2, $3, $4)`, c.IDCategoria, c.Nombre, c.Descripcion, c.Activo)
	return err
}

func (r *repository) ActualizarCategoria(ctx context.Context, c *Categoria) (bool, error) {
	res, err := r.db.ExecContext(ctx, `
		UPDATE categorias_producto SET nombre = $1, descripcion = $2, activo = $3
		WHERE id_categoria = $4`, c.Nombre, c.Descripcion, c.Activo, c.IDCategoria)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n > 0, err
}

const selectProducto = `
	SELECT p.id_producto, p.id_categoria, c.nombre, p.codigo, p.nombre, p.descripcion,
		p.precio_venta, p.stock, p.stock_minimo, p.activo
	FROM productos p
	INNER JOIN categorias_producto c ON c.id_categoria = p.id_categoria
`

type scanner interface {
	Scan(dest ...any) error
}

func scanProducto(sc scanner, p *Producto) error {
	if err := sc.Scan(
		&p.IDProducto, &p.IDCategoria, &p.Categoria, &p.Codigo, &p.Nombre, &p.Descripcion,
		&p.PrecioVenta, &p.Stock, &p.StockMinimo, &p.Activo,
	); err != nil {
		return err
	}
	p.StockBajo = p.Stock <= p.StockMinimo
	return nil
}

func (r *repository) ListarProductos(ctx context.Context, f FiltroProductos) ([]Producto, error) {
	rows, err := r.db.QueryContext(ctx, selectProducto+`
		WHERE ($1::boolean = false OR (p.activo AND c.activo))
		  AND ($2::uuid IS NULL OR p.id_categoria = $2::uuid)
		  AND ($3 = '' OR p.nombre ILIKE '%' || $3 || '%' OR p.codigo ILIKE '%' || $3 || '%')
		  AND ($4::boolean = false OR p.stock <= p.stock_minimo)
		ORDER BY c.nombre, p.nombre`,
		f.SoloActivos, f.IDCategoria, f.Busqueda, f.SoloBajos)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	lista := []Producto{}
	for rows.Next() {
		var p Producto
		if err := scanProducto(rows, &p); err != nil {
			return nil, err
		}
		lista = append(lista, p)
	}
	return lista, rows.Err()
}

func (r *repository) ObtenerProducto(ctx context.Context, id uuid.UUID) (*Producto, error) {
	var p Producto
	err := scanProducto(r.db.QueryRowContext(ctx, selectProducto+` WHERE p.id_producto = $1`, id), &p)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &p, nil
}

func (r *repository) CrearProducto(ctx context.Context, p *Producto) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO productos (
			id_producto, id_categoria, codigo, nombre, descripcion,
			precio_venta, stock, stock_minimo, activo
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		p.IDProducto, p.IDCategoria, p.Codigo, p.Nombre, p.Descripcion,
		p.PrecioVenta, p.Stock, p.StockMinimo, p.Activo)
	return err
}

// ActualizarProducto no toca el stock: este solo cambia con AjustarStock o con pedidos.
func (r *repository) ActualizarProducto(ctx context.Context, p *Producto) (bool, error) {
	res, err := r.db.ExecContext(ctx, `
		UPDATE productos
		SET id_categoria = $1, codigo = $2, nombre = $3, descripcion = $4,
			precio_venta = $5, stock_minimo = $6, activo = $7
		WHERE id_producto = $8`,
		p.IDCategoria, p.Codigo, p.Nombre, p.Descripcion,
		p.PrecioVenta, p.StockMinimo, p.Activo, p.IDProducto)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n > 0, err
}

// AjustarStock aplica delta de forma atómica; devuelve false si el producto no
// existe o si el resultado sería negativo.
func (r *repository) AjustarStock(ctx context.Context, id uuid.UUID, delta int) (bool, error) {
	res, err := r.db.ExecContext(ctx, `
		UPDATE productos SET stock = stock + $1
		WHERE id_producto = $2 AND stock + $1 >= 0`, delta, id)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n > 0, err
}
