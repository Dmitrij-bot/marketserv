package usecase

import (
	"context"
	"fmt"
	"github.com/Dmitrij-bot/marketserv/internal/repository"
	"github.com/avito-tech/go-transaction-manager/trm"
	"log"
)

type txManager interface {
	Do(ctx context.Context, f func(ctx context.Context) error) error
}

type UserUseCase struct {
	r         repository.Interface
	txManager txManager
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

func New(r repository.Interface, txManager trm.Manager) *UserUseCase {
	return &UserUseCase{
		r:         r,
		txManager: txManager,
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

	return SearchProductByNameResponse{
		Products: products,
	}, nil
}

func (u *UserUseCase) AddItemToCart(ctx context.Context, req AddItemToCartRequest) (resp AddItemToCartResponse, err error) {

	createCartResp, err := u.r.CreateCartIfNotExists(
		ctx,
		repository.CreateCartIfNotExistsRequest{
			ClientId: req.ClientId,
		})

	if err != nil {
		return AddItemToCartResponse{Success: false}, fmt.Errorf("failed to get CartId: %w", err)
	}

	req.CartId = createCartResp.CartId

	err = u.txManager.Do(ctx, func(ctx context.Context) error {

		addResp, err := u.r.AddItemToCart(
			ctx,
			repository.AddItemToCartRequest{
				CartId:    req.CartId,
				ProductID: req.ProductID,
				Quantity:  req.Quantity,
				ClientId:  req.ClientId,
			})

		if err != nil {
			return fmt.Errorf("failed to add to cart: %w", err)
		}

		addEvent := AddEvent{
			ClientID:  int(req.ClientId),
			ProductID: req.ProductID,
			Quantity:  req.Quantity,
			Message: fmt.Sprintf("Товар успешно добавлен в корзину {\"client_id\":%d,\"product_id\":%d,\"quantity\":%d}",
				req.ClientId, req.ProductID, req.Quantity),
		}

		saveResp, err := u.r.SaveKafkaMessage(
			ctx,
			repository.SaveKafkaMessageRequest{
				KafkaMessage: addEvent,
				KafkaKey:     AddEventKey,
			})
		if err != nil {
			log.Printf("Ошибка сохранения сообщения: %v", err)
			return fmt.Errorf("failed to save Kafka message: %w", err)
		} else {
			log.Printf("событие сохранено: %v", saveResp)
		}

		getPriceResp, err := u.r.GetItemPrice(ctx, repository.GetItemPriceRequest{
			ProductID: req.ProductID,
		})

		if err != nil {
			return fmt.Errorf("failed to get product price: %w", err)
		}

		getRedis, err := u.r.GetCartFromRedis(
			ctx,
			repository.GetCartFromRedisRequest{
				ClientId: req.ClientId,
			})
		if err != nil {
			log.Printf("Ошибка получения корзины из Redis для ClientId %d: %v", req.ClientId, err)
		} else {
			log.Printf("Получена корзина из Redis для ClientId %d: %v", req.ClientId, getRedis)
		}

		if len(getRedis.CartItems) > 0 {
			fmt.Println("Cart Items:")
			for _, item := range getRedis.CartItems {
				fmt.Printf("Product ID: %d, Quantity: %d, Price: %.2f\n", item.ProductID, item.ProductQuantity, item.ProductPrice)
			}
		} else {
			fmt.Println("No items in the cart.")
		}

		getRedis.CartItems = updateCartWithItem(getRedis.CartItems, req.ProductID, req.Quantity, getPriceResp.ProductPrice)

		_, err = u.r.SetCartInRedis(
			ctx,
			repository.SetCartInRedisRequest{
				RedisKey:  getRedis.RedisKey,
				CartItems: getRedis.CartItems,
			})
		if err != nil {
			log.Printf("Ошибка сохранения корзины в Redis: %v", err)
			return fmt.Errorf("failed to save updated cart to Redis: %w", err)
		}

		resp.Success = addResp.Success
		return nil
	})

	if err != nil {
		return AddItemToCartResponse{Success: false}, err
	}

	return resp, nil
}

func updateCartWithItem(cartItems []repository.CartItem, productID int32, quantity int32, productPrice float64) []repository.CartItem {
	for i, item := range cartItems {

		if item.ProductID == productID {
			if quantity == 0 {
				// Если quantity == 0, уменьшаем количество на 1 (удаляем товар)
				cartItems[i].ProductQuantity--
				if cartItems[i].ProductQuantity <= 0 {
					// Если количество товара стало 0 или меньше, удаляем его из корзины
					cartItems = append(cartItems[:i], cartItems[i+1:]...)
				}
			} else {
				// Если quantity > 0, обновляем количество товара
				cartItems[i].ProductQuantity += quantity
			}
			return cartItems
		}
	}

	if quantity > 0 {
		newItem := repository.CartItem{
			ProductID:       productID,
			ProductQuantity: quantity,
			ProductPrice:    productPrice,
		}
		return append(cartItems, newItem)
	}

	return cartItems
}

func (u *UserUseCase) DeleteItemFromCart(ctx context.Context, req DeleteItemFromCartRequest) (resp DeleteItemFromCartResponse, err error) {

	createCartResp, err := u.r.CreateCartIfNotExists(
		ctx,
		repository.CreateCartIfNotExistsRequest{
			ClientId: req.ClientId,
		})

	if err != nil {
		return DeleteItemFromCartResponse{Success: false}, fmt.Errorf("failed to get CartId: %w", err)
	}

	req.CartId = createCartResp.CartId

	err = u.txManager.Do(ctx, func(ctx context.Context) error {

		deleteResp, err := u.r.DeleteItemFromCart(
			ctx,
			repository.DeleteItemFromCartRequest{
				CartId:    req.CartId,
				ClientId:  req.ClientId,
				ProductID: req.ProductID,
			})
		if err != nil {
			return fmt.Errorf("failed to delete from cart: %w", err)
		}

		deleteEvent := DeleteEvent{
			ClientID: int(req.ClientId),
			Message: fmt.Sprintf("Товар успешно удален из корзины {\"client_id\":%d,\"product_id\":%d}",
				req.ClientId, req.ProductID),
		}

		saveResp, err := u.r.SaveKafkaMessage(
			ctx,
			repository.SaveKafkaMessageRequest{
				KafkaMessage: deleteEvent,
				KafkaKey:     DeleteEventKey,
			})
		if err != nil {
			log.Printf("Ошибка сохранения сообщения: %v", err)
			return fmt.Errorf("failed to save Kafka message: %w", err)
		} else {
			log.Printf("событие сохранено: %v", saveResp)
		}

		getRedis, err := u.r.GetCartFromRedis(
			ctx,
			repository.GetCartFromRedisRequest{
				ClientId: req.ClientId,
			})
		if err != nil {
			log.Printf("Ошибка получения корзины из Redis для ClientId %d: %v", req.ClientId, err)
		} else {
			log.Printf("Получена корзина из Redis для ClientId %d: %v", req.ClientId, getRedis)
		}

		if len(getRedis.CartItems) > 0 {
			fmt.Println("Cart Items:")
			for _, item := range getRedis.CartItems {
				fmt.Printf("Product ID: %d, Quantity: %d, Price: %.2f\n", item.ProductID, item.ProductQuantity, item.ProductPrice)
			}
		} else {
			fmt.Println("No items in the cart.")
		}

		getRedis.CartItems = updateCartWithItem(getRedis.CartItems, req.ProductID, 0, 0)

		_, err = u.r.SetCartInRedis(
			ctx,
			repository.SetCartInRedisRequest{
				RedisKey:  getRedis.RedisKey,
				CartItems: getRedis.CartItems,
			})
		if err != nil {
			log.Printf("Ошибка сохранения корзины в Redis: %v", err)
			return fmt.Errorf("failed to save updated cart to Redis: %w", err)
		}

		resp.Success = deleteResp.Success
		return nil
	})

	if err != nil {
		return DeleteItemFromCartResponse{Success: false}, err
	}

	return resp, nil
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
