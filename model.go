package pgparty

type Model struct {
	ID        UUIDv4   `json:"id"`
	CreatedAt Time     `json:"createdAt"`
	UpdatedAt Time     `json:"updatedAt"`
	DeletedAt NullTime `json:"-" key:"deleted_at_idx"`
}

func (m *Model) IsIDEmpty() bool {
	return m.ID.IsZero()
}

func (m *Model) MarkUpdated() {
	if m.CreatedAt.Time().IsZero() {
		m.CreatedAt = NowUTC()
	}
	m.UpdatedAt = NowUTC()
}

type Model58 struct {
	ID        UUID58   `json:"id"`
	CreatedAt Time     `json:"createdAt"`
	UpdatedAt Time     `json:"updatedAt"`
	DeletedAt NullTime `json:"-" key:"deleted_at_idx"`
}

func (m *Model58) IsIDEmpty() bool {
	return m.ID.IsZero()
}

func (m *Model58) MarkUpdated() {
	if m.CreatedAt.Time().IsZero() {
		m.CreatedAt = NowUTC()
	}
	m.UpdatedAt = NowUTC()
}
