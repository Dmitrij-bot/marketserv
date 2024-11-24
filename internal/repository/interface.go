package repository

import "context"

type Interface interface {
	FindClientByUsername(ctx context.Context, req FindClientByUsernameRequest) (resp FindClientByUsernameResponse, err error)
	SearchProductByName(ctx context.Context, req SearchProductByNameRequest) (resp SearchProductByNameResponse, err error)
	CreateCartIfNotExists(ctx context.Context, req CreateCartIfNotExistsRequest) (resp CreateCartIfNotExistsResponse, err error)
	AddItemToCart(ctx context.Context, req AddItemToCartRequest) (resp AddItemToCartResponse, err error)
	DeleteItemFromCart(ctx context.Context, req DeleteItemFromCartRequest) (resp DeleteItemFromCartResponse, err error)
	GetCart(ctx context.Context, req GetCartRequest) (resp GetCartResponse, err error)
	SimulatePayment(ctx context.Context, req PaymentRequest) (resp PaymentResponse, err error)
	SaveKafkaMessage(ctx context.Context, req SaveKafkaMessageRequest) (resp SaveKafkaMessageResponse, err error)
	GetKafkaMessage() (resp GetKafkaMessageResponse, err error)
	SetDone(id int) error
}
