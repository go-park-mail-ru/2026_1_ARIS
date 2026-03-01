package repository

type Repository struct {
	UserRepo UserRepo
	PostRepo PostRepo
}

func NewRepository() *Repository {
	return &Repository{
		UserRepo: &inmemoryUserRepo{},
		PostRepo: &inmemoryPostRepo{},
	}
}
