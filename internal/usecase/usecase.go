package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/Dmitrij-bot/marketserv/internal/repository"
	"github.com/IBM/sarama"
	"log"
)

type UserUseCase struct {
	r repository.Interface
}

type PaymentEvent struct {
	ClientID int    `json:"client_id"`
	Status   string `json:"status"`
	Message  string `json:"message"`
}

type GetCartEvent struct {
	ClientID   int        `json:"client_id"`
	CartItems  []CartItem `json:"cart_items"`
	TotalPrice string     `json:"total_price"`
	Message    string     `json:"message"`
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

	if addResp.Success {

		message := fmt.Sprintf("Товар успешно добавлен в корзину {\"client_id\":%d,\"product_id\":%d,\"quantity\":%d}",
			req.ClientId, req.ProductID, req.Quantity)

		err := u.sendKafkaMessage(message, "InsufficientBalance")
		if err != nil {
			log.Printf("Ошибка отправки сообщения в Kafka: %v", err)
		} else {
			log.Printf("Событие отправлено в Kafka: %v", req)
		}
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
		return DeleteItemFromCartResponse{Success: false}, fmt.Errorf("failed to delete from cart: %w", err)
	}

	if deleteResp.Success {

		message := fmt.Sprintf("Товар успешно удален из корзины {\"client_id\":%d,\"product_id\":%d}",
			req.ClientId, req.ProductID)

		err := u.sendKafkaMessage(message, "InsufficientBalance")
		if err != nil {
			log.Printf("Ошибка отправки сообщения в Kafka: %v", err)
		} else {
			log.Printf("Событие отправлено в Kafka: %v", req)
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

	err = u.sendKafkaMessage(cartEvent, "GetCartTrue")
	if err != nil {
		log.Printf("Ошибка отправки сообщения в Kafka: %v", err)
	} else {
		log.Printf("Событие отправлено в Kafka: %v", cartEvent)
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
				Status:   "insufficient_balance",
				Message:  fmt.Sprintf("Недостаточно средств для клиента %d", req.ClientId),
			}

			sendErr := u.sendKafkaMessage(paymentEvent, "InsufficientBalance")
			if sendErr != nil {
				log.Printf("Ошибка отправки сообщения в Kafka: %v", sendErr)
			} else {
				log.Printf("Событие отправлено в Kafka: %v", paymentEvent)
			}
		}
		return PaymentResponse{Success: false}, fmt.Errorf("ошибка выполнения платежа: %w", err)
	}

	if paymentResp.Success {
		paymentEvent := PaymentEvent{
			ClientID: int(req.ClientId),
			Status:   "payment_success",
			Message:  fmt.Sprintf("Товар успешно оплачен клиентом %d", req.ClientId),
		}
		sendErr := u.sendKafkaMessage(paymentEvent, "InsufficientBalance")
		if sendErr != nil {
			log.Printf("Ошибка отправки сообщения в Kafka: %v", sendErr)
		} else {
			log.Printf("Событие отправлено в Kafka: %v", paymentEvent)
		}
	}

	return PaymentResponse{
		Success: paymentResp.Success,
	}, nil
}

func (u *UserUseCase) sendKafkaMessage(event interface{}, key string) error {

	config := sarama.NewConfig()
	config.Producer.Return.Successes = true

	producer, err := sarama.NewSyncProducer([]string{"localhost:29092"}, config)
	if err != nil {
		return fmt.Errorf("ошибка создания Kafka producer: %w", err)
	}
	defer producer.Close()

	messageBytes, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("ошибка сериализации сообщения: %w", err)
	}

	msg := &sarama.ProducerMessage{
		Topic: "test1",
		//Key:   sarama.StringEncoder("InsufficientBalance"),
		Key:   sarama.StringEncoder(key),
		Value: sarama.StringEncoder(string(messageBytes)),
	}

	partition, offset, err := producer.SendMessage(msg)
	if err != nil {
		return fmt.Errorf("ошибка отправки сообщения в Kafka: %w", err)
	}

	log.Printf("Сообщение успешно отправлено в Kafka: partition=%d, offset=%d", partition, offset)
	return nil
}
