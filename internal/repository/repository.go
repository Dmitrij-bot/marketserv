package repository

import (
	"context"
	"database/sql"
)

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{
		db: db,
	}
}

func (r *UserRepository) FindClientByUsername(ctx context.Context, req FindClientByUsernameRequest) (resp FindClientByUsernameResponse, err error) {
	err = r.db.QueryRowContext(ctx, FindClientByUserNameSql, req.ClientID).Scan(&resp.ClientID, &resp.Username, &resp.Role)
	if err != nil {
		return resp, err
	}
	return resp, nil
}

func (r *UserRepository) SearchProductByName(ctx context.Context, req SearchProductByNameRequest) (resp SearchProductByNameResponse, err error) {
	err = r.db.QueryRowContext(ctx, SearchProductByNameSQL, req.ProductName).Scan(&resp.ProductID, &resp.ProductName, &resp.ProductDescription, &resp.ProductPrice)
	if err != nil {
		return resp, err
	}
	return resp, nil
}
