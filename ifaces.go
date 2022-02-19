package pgparty

// Storable - интерфейс, которая должна реализовывать структура, которая должна храниться в базе.
type Storable interface {
	StoreName() string
}
