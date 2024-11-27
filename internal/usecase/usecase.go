package usecase

import (
	"context"
	"fmt"
	"github.com/Dmitrij-bot/marketserv/internal/repository"
	"log"
	"strconv"
)

type UserUseCase struct {
	r repository.Interface
}

type SearchProductEvent struct {
	ProductName string    `json:"product_name"`
	Products    []Product `json:"products"`
	Message     string    `json:"message"`
}

type AddEvent struct {
	ClientID  int    `json:"client_id"`
	ProductID int32  `json:"product_id" db:"product_id"`
	Quantity  int32  `json:"quantity" db:"quantity"`
	Message   string `json:"message"`
}

type DeleteEvent struct {
	ClientID int    `json:"client_id"`
	Message  string `json:"message"`
}

type GetCartEvent struct {
	ClientID   int        `json:"client_id"`
	CartItems  []CartItem `json:"cart_items"`
	TotalPrice string     `json:"total_price"`
	Message    string     `json:"message"`
}

type PaymentEvent struct {
	ClientID int    `json:"client_id"`
	Status   string `json:"status"`
	Message  string `json:"message"`
}

func New(r repository.Interface) *UserUseCase {
	return &UserUseCase{
		r: r,
	}
}

func (u *UserUseCase) FindClientByUsername(ctx context.Context, req FindClientByUsernameRequest) (resp FindClientByUsernameResponse, err error) {

	user, err := u.r.FindClientByUsername(
		ctx,
		repository.FindClientByUsernameRequest{
			ClientID: req.ClientID,
		})
	if err != nil {
		return resp, err
	}

	return FindClientByUsernameResponse{
		ClientID: user.ClientID,
		Username: user.Username,
		Role:     user.Role,
	}, nil
}

func (u *UserUseCase) SearchProductByName(ctx context.Context, req SearchProductByNameRequest) (resp SearchProductByNameResponse, err error) {

	if req.ProductName == "" {
		return SearchProductByNameResponse{}, fmt.Errorf("product name cannot be empty")
	}

	productsResp, err := u.r.SearchProductByName(
		ctx,
		repository.SearchProductByNameRequest{
			ProductName: req.ProductName,
		})
	if err != nil {
		return SearchProductByNameResponse{}, fmt.Errorf("failed to search products: %w", err)
	}

	var products []Product
	for _, p := range productsResp.Products {
		products = append(products, Product{
			ProductID:          p.ProductID,
			ProductName:        p.ProductName,
			ProductDescription: p.ProductDescription,
			ProductPrice:       p.ProductPrice,
		})
	}

	if len(products) == 0 {

		emptySearchEvent := SearchProductEvent{
			ProductName: req.ProductName,
			Products:    nil,
			Message:     fmt.Sprintf("Продукты по запросу '%s' не найдены", req.ProductName),
		}

		saveEvent, err := u.r.SaveKafkaMessage(
			ctx,
			repository.SaveKafkaMessageRequest{
				KafkaMessage: emptySearchEvent,
				KafkaKey:     "SearchProductFalse",
			})

		if err != nil {
			log.Printf("Ошибка сохранения сообщения: %v", err)
		} else {
			log.Printf("событие сохранено: %v", saveEvent)
		}

		return SearchProductByNameResponse{}, fmt.Errorf("продукты по запросу '%s' не найдены", req.ProductName)
	}

	searchEvent := SearchProductEvent{
		ProductName: req.ProductName,
		Products:    products,
		Message:     fmt.Sprintf("Продукты по запросу '%s' успешно найдены", req.ProductName),
	}

	saveEvent, err := u.r.SaveKafkaMessage(
		ctx,
		repository.SaveKafkaMessageRequest{
			KafkaMessage: searchEvent,
			KafkaKey:     "SearchProductTrue",
		})

	if err != nil {
		log.Printf("Ошибка сохранения сообщения: %v", err)
	} else {
		log.Printf("событие сохранено: %v", saveEvent)
	}

	return SearchProductByNameResponse{
		Products: products,
	}, nil
}

func (u *UserUseCase) AddItemToCart(ctx context.Context, req AddItemToCartRequest) (resp AddItemToCartResponse, err error) {

	addResp, err := u.r.AddItemToCart(
		ctx,
		repository.AddItemToCartRequest{
			CartId:    req.CartId,
			ProductID: req.ProductID,
			Quantity:  req.Quantity,
			ClientId:  req.ClientId,
		})

	if err != nil {

		return AddItemToCartResponse{Success: false}, fmt.Errorf("failed to add to cart: %w", err)

	}

	return AddItemToCartResponse{
		Success: addResp.Success,
	}, nil
}

func (u *UserUseCase) DeleteItemFromCart(ctx context.Context, req DeleteItemFromCartRequest) (resp DeleteItemFromCartResponse, err error) {

	deleteResp, err := u.r.DeleteItemFromCart(
		ctx,
		repository.DeleteItemFromCartRequest{
			ClientId:  req.ClientId,
			ProductID: req.ProductID,
		})
	if err != nil {
		deleteEvent := DeleteEvent{
			ClientID: int(req.ClientId),
			Message: fmt.Sprintf("Ошибка удаления товара из корзины {\"client_id\":%d,\"product_id\":%d}",
				req.ClientId, req.ProductID),
		}

		saveEvent, err := u.r.SaveKafkaMessage(
			ctx,
			repository.SaveKafkaMessageRequest{
				KafkaMessage: deleteEvent,
				KafkaKey:     "DeleteItemFalse",
			})

		if err != nil {
			log.Printf("Ошибка сохранения сообщения: %v", err)
		} else {
			log.Printf("событие сохранено: %v", saveEvent)
		}

		return DeleteItemFromCartResponse{Success: false}, fmt.Errorf("failed to delete from cart: %w", err)
	}

	if deleteResp.Success {

		deleteEvent := DeleteEvent{
			ClientID: int(req.ClientId),
			Message: fmt.Sprintf("Товар успешно удален из корзины {\"client_id\":%d,\"product_id\":%d}",
				req.ClientId, req.ProductID),
		}

		saveEvent, err := u.r.SaveKafkaMessage(
			ctx,
			repository.SaveKafkaMessageRequest{
				KafkaMessage: deleteEvent,
				KafkaKey:     "DeleteItemTrue",
			})

		if err != nil {
			log.Printf("Ошибка сохранения сообщения: %v", err)
		} else {
			log.Printf("событие сохранено: %v", saveEvent)
		}
	}

	return DeleteItemFromCartResponse{
		Success: deleteResp.Success,
	}, nil
}

func (u *UserUseCase) GetCart(ctx context.Context, req GetCartRequest) (resp GetCartResponse, err error) {

	if req.ClientId == 0 {
		return GetCartResponse{}, fmt.Errorf("invalid user_id: %d", req.ClientId)
	}

	getResp, err := u.r.GetCart(
		ctx,
		repository.GetCartRequest{
			ClientId: req.ClientId})
	if err != nil {

		cartErrorEvent := GetCartEvent{
			ClientID:   int(req.ClientId),
			CartItems:  nil,
			TotalPrice: strconv.Itoa(0),
			Message:    fmt.Sprintf("Ошибка получения корзины для клиента %d: %v", req.ClientId, err),
		}

		saveEvent, err := u.r.SaveKafkaMessage(
			ctx,
			repository.SaveKafkaMessageRequest{
				KafkaMessage: cartErrorEvent,
				KafkaKey:     "GetCartFalse",
			})

		if err != nil {
			log.Printf("Ошибка сохранения сообщения: %v", err)
		} else {
			log.Printf("событие сохранено: %v", saveEvent)
		}

		return GetCartResponse{}, fmt.Errorf("usecase: failed to get cart for user_id %d: %v", req.ClientId, err)
	}

	var cartItems []CartItem
	for _, repoItem := range getResp.CartItems {
		cartItems = append(cartItems, CartItem{
			ProductID:       repoItem.ProductID,
			ProductQuantity: repoItem.ProductQuantity,
			ProductPrice:    repoItem.ProductPrice,
		})
	}

	cartEvent := GetCartEvent{
		ClientID:   int(req.ClientId),
		CartItems:  cartItems,
		TotalPrice: getResp.TotalPrice,
		Message:    fmt.Sprintf("Корзина для клиента %d успешно получена", req.ClientId),
	}

	saveEvent, err := u.r.SaveKafkaMessage(
		ctx,
		repository.SaveKafkaMessageRequest{
			KafkaMessage: cartEvent,
			KafkaKey:     "GetCartTrue",
		})

	if err != nil {
		log.Printf("Ошибка сохранения сообщения: %v", err)
	} else {
		log.Printf("событие сохранено: %v", saveEvent)
	}

	return GetCartResponse{
		CartItems:  cartItems,
		TotalPrice: getResp.TotalPrice,
	}, nil
}

func (u *UserUseCase) SimulatePayment(ctx context.Context, req PaymentRequest) (resp PaymentResponse, err error) {

	paymentResp, err := u.r.SimulatePayment(
		ctx,
		repository.PaymentRequest{
			ClientId: req.ClientId,
		})
	if err != nil {
		if err.Error() == "платёж не выполнен: возможно, недостаточно средств на счёте" {
			paymentEvent := PaymentEvent{
				ClientID: int(req.ClientId),
				Message:  fmt.Sprintf("Недостаточно средств для клиента %d", req.ClientId),
			}

			saveEvent, err := u.r.SaveKafkaMessage(
				ctx,
				repository.SaveKafkaMessageRequest{
					KafkaMessage: paymentEvent,
					KafkaKey:     "SimulatePaymentFalse",
				})

			if err != nil {
				log.Printf("Ошибка сохранения сообщения: %v", err)
			} else {
				log.Printf("событие сохранено: %v", saveEvent)
			}
		}
		return PaymentResponse{Success: false}, fmt.Errorf("ошибка выполнения платежа: %w", err)
	}

	if paymentResp.Success {
		paymentEvent := PaymentEvent{
			ClientID: int(req.ClientId),
			Message:  fmt.Sprintf("Товар успешно оплачен клиентом %d", req.ClientId),
		}

		saveEvent, err := u.r.SaveKafkaMessage(
			ctx,
			repository.SaveKafkaMessageRequest{
				KafkaMessage: paymentEvent,
				KafkaKey:     "SimulatePaymentTrue",
			})

		if err != nil {
			log.Printf("Ошибка сохранения сообщения: %v", err)
		} else {
			log.Printf("событие сохранено: %v", saveEvent)
		}

	}

	return PaymentResponse{
		Success: paymentResp.Success,
	}, nil
}
