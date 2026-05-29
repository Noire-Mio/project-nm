package repositories

type ProductFactory func(ctx *GormDBContext) IProduct


type IProduct interface {
}

type ProductRepository struct {
	GormRepository
}

func NewProductRepo(ctx *GormDBContext) IProduct {
	repository := new(ProductRepository)
	repository.SetDBContext(ctx)
	return repository
}
