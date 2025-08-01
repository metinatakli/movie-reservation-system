package domain

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type Cart struct {
	Id          string `json:"-"`
	ShowtimeID  int
	TotalPrice  decimal.Decimal
	BasePrice   decimal.Decimal
	MovieName   string
	TheaterName string
	HallName    string
	Date        time.Time
	Seats       []CartSeat
}

type CartSeat struct {
	Id         int
	Row        int
	Col        int
	SeatType   string
	ExtraPrice decimal.Decimal
}

func NewCart(showtimeID int, showtimeSeats *ShowtimeSeats) Cart {
	id := uuid.New().String()
	seats := toCartSeats(showtimeSeats.Seats)
	basePrice := decimal.NewFromFloat(showtimeSeats.Price)
	totalPrice := calculateTotalPrice(basePrice, seats)

	return Cart{
		Id:          id,
		ShowtimeID:  showtimeID,
		TotalPrice:  totalPrice,
		BasePrice:   basePrice,
		MovieName:   showtimeSeats.MovieName,
		TheaterName: showtimeSeats.TheaterName,
		HallName:    showtimeSeats.HallName,
		Date:        showtimeSeats.Date,
		Seats:       seats,
	}
}

func calculateTotalPrice(basePrice decimal.Decimal, cartSeats []CartSeat) decimal.Decimal {
	total := decimal.Zero

	for _, v := range cartSeats {
		seatPrice := basePrice.Add(v.ExtraPrice)
		total = total.Add(seatPrice)
	}

	return total
}

func toCartSeats(seats []Seat) []CartSeat {
	cartSeats := make([]CartSeat, len(seats))

	for i, seat := range seats {
		cartSeat := CartSeat{
			Id:       seat.ID,
			Row:      seat.Row,
			Col:      seat.Col,
			SeatType: seat.Type,
		}

		priceFloat := seat.ExtraPrice
		cartSeat.ExtraPrice = decimal.NewFromFloat(priceFloat)

		cartSeats[i] = cartSeat
	}

	return cartSeats
}
