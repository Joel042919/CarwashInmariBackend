package productos

import (
	"context"
	"math"
	"strings"

	"github.com/google/uuid"

	"carwashinmaribackend/internal/utils"
)

type Service interface {
	ListarCategorias(ctx context.Context, soloActivas bool) ([]Categoria, error)
	CrearCategoria(ctx context.Context, in CategoriaInput) (*Categoria, error)
	ActualizarCategoria(ctx context.Context, id uuid.UUID, in CategoriaInput) (*Categoria, error)

	ListarProductos(ctx context.Context, f FiltroProductos) ([]Producto, error)
	ObtenerProducto(ctx context.Context, id uuid.UUID, soloActivos bool) (*Producto, error)
	CrearProducto(ctx context.Context, in ProductoInput) (*Producto, error)
	ActualizarProducto(ctx context.Context, id uuid.UUID, in ProductoInput) (*Producto, error)
	AjustarStock(ctx context.Context, id uuid.UUID, delta int) (*Producto, error)
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func limpiarOpcional(s *string) *string {
	if s == nil {
		return nil
	}
	t := strings.TrimSpace(*s)
	if t == "" {
		return nil
	}
	return &t
}

func validarCategoria(in *CategoriaInput) error {
	in.Nombre = strings.TrimSpace(in.Nombre)
	if in.Nombre == "" || len(in.Nombre) > 60 {
		return utils.BadRequest("el nombre de la categoría es obligatorio (máximo 60 caracteres)")
	}
	in.Descripcion = limpiarOpcional(in.Descripcion)
	if in.Descripcion != nil && len(*in.Descripcion) > 200 {
		return utils.BadRequest("la descripción no puede superar 200 caracteres")
	}
	return nil
}

func (s *service) ListarCategorias(ctx context.Context, soloActivas bool) ([]Categoria, error) {
	return s.repo.ListarCategorias(ctx, soloActivas)
}

func (s *service) CrearCategoria(ctx context.Context, in CategoriaInput) (*Categoria, error) {
	if err := validarCategoria(&in); err != nil {
		return nil, err
	}
	activo := true
	if in.Activo != nil {
		activo = *in.Activo
	}
	c := &Categoria{IDCategoria: uuid.New(), Nombre: in.Nombre, Descripcion: in.Descripcion, Activo: activo}
	if err := s.repo.CrearCategoria(ctx, c); err != nil {
		if utils.IsUniqueViolation(err) {
			return nil, utils.Conflict("ya existe una categoría con ese nombre")
		}
		return nil, err
	}
	return c, nil
}

func (s *service) ActualizarCategoria(ctx context.Context, id uuid.UUID, in CategoriaInput) (*Categoria, error) {
	if err := validarCategoria(&in); err != nil {
		return nil, err
	}
	activo := true
	if in.Activo != nil {
		activo = *in.Activo
	}
	c := &Categoria{IDCategoria: id, Nombre: in.Nombre, Descripcion: in.Descripcion, Activo: activo}
	ok, err := s.repo.ActualizarCategoria(ctx, c)
	if err != nil {
		if utils.IsUniqueViolation(err) {
			return nil, utils.Conflict("ya existe una categoría con ese nombre")
		}
		return nil, err
	}
	if !ok {
		return nil, utils.NotFound("categoría no encontrada")
	}
	return c, nil
}

func (s *service) ListarProductos(ctx context.Context, f FiltroProductos) ([]Producto, error) {
	f.Busqueda = strings.TrimSpace(f.Busqueda)
	return s.repo.ListarProductos(ctx, f)
}

func (s *service) ObtenerProducto(ctx context.Context, id uuid.UUID, soloActivos bool) (*Producto, error) {
	p, err := s.repo.ObtenerProducto(ctx, id)
	if err != nil {
		return nil, err
	}
	if p == nil || (soloActivos && !p.Activo) {
		return nil, utils.NotFound("producto no encontrado")
	}
	return p, nil
}

func (s *service) validarProducto(ctx context.Context, in *ProductoInput) error {
	in.Codigo = strings.TrimSpace(in.Codigo)
	in.Nombre = strings.TrimSpace(in.Nombre)
	if in.Codigo == "" || len(in.Codigo) > 30 {
		return utils.BadRequest("el código es obligatorio (máximo 30 caracteres)")
	}
	if in.Nombre == "" || len(in.Nombre) > 100 {
		return utils.BadRequest("el nombre es obligatorio (máximo 100 caracteres)")
	}
	in.Descripcion = limpiarOpcional(in.Descripcion)
	if in.PrecioVenta <= 0 {
		return utils.BadRequest("el precio debe ser mayor a 0")
	}
	in.PrecioVenta = math.Round(in.PrecioVenta*100) / 100
	if in.Stock < 0 {
		return utils.BadRequest("el stock no puede ser negativo")
	}
	if in.StockMinimo != nil && *in.StockMinimo < 0 {
		return utils.BadRequest("el stock mínimo no puede ser negativo")
	}
	if in.IDCategoria == uuid.Nil {
		return utils.BadRequest("la categoría es obligatoria")
	}
	existe, err := s.repo.ExisteCategoria(ctx, in.IDCategoria)
	if err != nil {
		return err
	}
	if !existe {
		return utils.BadRequest("la categoría indicada no existe")
	}
	return nil
}

func (s *service) CrearProducto(ctx context.Context, in ProductoInput) (*Producto, error) {
	if err := s.validarProducto(ctx, &in); err != nil {
		return nil, err
	}
	stockMin, activo := 5, true
	if in.StockMinimo != nil {
		stockMin = *in.StockMinimo
	}
	if in.Activo != nil {
		activo = *in.Activo
	}
	p := &Producto{
		IDProducto:  uuid.New(),
		IDCategoria: in.IDCategoria,
		Codigo:      in.Codigo,
		Nombre:      in.Nombre,
		Descripcion: in.Descripcion,
		PrecioVenta: in.PrecioVenta,
		Stock:       in.Stock,
		StockMinimo: stockMin,
		Activo:      activo,
	}
	if err := s.repo.CrearProducto(ctx, p); err != nil {
		if utils.IsUniqueViolation(err) {
			return nil, utils.Conflict("ya existe un producto con ese código")
		}
		return nil, err
	}
	return s.repo.ObtenerProducto(ctx, p.IDProducto)
}

func (s *service) ActualizarProducto(ctx context.Context, id uuid.UUID, in ProductoInput) (*Producto, error) {
	if err := s.validarProducto(ctx, &in); err != nil {
		return nil, err
	}
	actual, err := s.repo.ObtenerProducto(ctx, id)
	if err != nil {
		return nil, err
	}
	if actual == nil {
		return nil, utils.NotFound("producto no encontrado")
	}
	stockMin, activo := actual.StockMinimo, actual.Activo
	if in.StockMinimo != nil {
		stockMin = *in.StockMinimo
	}
	if in.Activo != nil {
		activo = *in.Activo
	}
	p := &Producto{
		IDProducto:  id,
		IDCategoria: in.IDCategoria,
		Codigo:      in.Codigo,
		Nombre:      in.Nombre,
		Descripcion: in.Descripcion,
		PrecioVenta: in.PrecioVenta,
		StockMinimo: stockMin,
		Activo:      activo,
	}
	if _, err := s.repo.ActualizarProducto(ctx, p); err != nil {
		if utils.IsUniqueViolation(err) {
			return nil, utils.Conflict("ya existe un producto con ese código")
		}
		return nil, err
	}
	return s.repo.ObtenerProducto(ctx, id)
}

func (s *service) AjustarStock(ctx context.Context, id uuid.UUID, delta int) (*Producto, error) {
	if delta == 0 {
		return nil, utils.BadRequest("la cantidad a ajustar no puede ser 0")
	}
	ok, err := s.repo.AjustarStock(ctx, id, delta)
	if err != nil {
		return nil, err
	}
	if !ok {
		p, err := s.repo.ObtenerProducto(ctx, id)
		if err != nil {
			return nil, err
		}
		if p == nil {
			return nil, utils.NotFound("producto no encontrado")
		}
		return nil, utils.BadRequest("el ajuste dejaría el stock en negativo")
	}
	return s.repo.ObtenerProducto(ctx, id)
}
