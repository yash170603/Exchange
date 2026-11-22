package orderbook

import (
	"errors"
	"fmt"
)

func InitializeOrderBook(Entity string) *OrderBook {
	orderBook := OrderBook{
		Entity:      Entity,
		buys:        []*Order{},
		sells:       []*Order{},
		nextSeq:     0,
		nextTradeID: 0,
	}

	return &orderBook
}

// order misses (assuming) seq,thats it.
func (orderBook *OrderBook) InsertOrder(orderReq *Order) error {
	if err := isOrderRquestValid(orderReq); err != nil {
		return errors.New("Error occured in orderValidation")
	}

	currentSeq := orderBook.nextSeq // extract current seq
	orderBook.nextTradeID++         // trade id for next is updated, not used yet

	currentSeq++ // add seq to order.
	orderReq.Seq = currentSeq
	orderBook.nextSeq++ // update seq in orderbook

	// add the order into orderbook.

	ordertype := orderReq.OrderType
	switch ordertype {
	case Buy:
		orderBook.buys = append(orderBook.buys, orderReq)
	case Sell:
		orderBook.sells = append(orderBook.sells, orderReq)
	default:
		fmt.Println("no OrderType present")

	}

	return nil

}

func (*OrderBook) CancelOrder(orderId int64) error {
	return nil
}

// for now its dummy i know.
func isOrderRquestValid(orderReq *Order) error {
	return nil
}
