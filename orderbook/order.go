package orderbook

type OrderType int

const (
	Buy OrderType = iota
	Sell
)

// OrderStatus is where an order is in its life.
//
//	Open ─► PartiallyFilled ─► Filled
//	  └───────────┴──────────► Cancelled

type OrderStatus int

const (
	OrderOpen OrderStatus = iota
	OrderPartiallyFilled
	OrderFilled
	OrderCancelled
	OrderRejected
)

type Order struct {
	ID        int64
	UserID    int64
	Entity    string // the stock, e.g. "RELIANCE"
	OrderType OrderType
	Price     int64  // limit price in paise: the worst price the trader accepts
	OrigQty   int64  // shares the trader asked for. Never changes
	Qty       int64  // shares still unfilled. Goes down as the order trades
	Seq       uint64 // arrival number, set by the book. Lower = arrived earlier
	Status    OrderStatus
}

// TradeStatus tracks a trade after it is created.
//
//	Executed ─► Saved
type TradeStatus int

const (
	TradeExecuted TradeStatus = iota
	TradeSaved
)

type Trade struct {
	ID          uint64
	Entity      string
	BuyOrderID  int64
	SellOrderID int64
	Price       int64
	Qty         int64
	Status      TradeStatus
}

type OrderBook struct {
	Entity      string
	buys        []*Order
	sells       []*Order
	nextSeq     uint64
	nextTradeID uint64
}
