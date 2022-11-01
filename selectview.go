package pgparty

import "strings"

type SelectView[T Storable] struct {
	// к каждому задействованному полю определяется способ выборки:
	// просто чтение поля с именем,
	// чтение полей суб-объекта из джойна,
	// выражение SQL
	Err error

	nmd *nodeMd
}

type nodeFd struct {
	parent *nodeMd
	fd     *FieldDescription
	child  *nodeMd // если поле это модель
}

type nodeMd struct {
	parent *nodeFd
	fields []*nodeFd
	md     *ModelDesc
}

func (nf *nodeMd) deepField(fdn string) (*nodeFd, error) {
	idx := strings.IndexByte(fdn, '.')
	fn := fdn
	if idx >= 0 {
		fn = fdn[:idx]
	}
	tail := fdn[idx+1:]



	
	curr := nf
	for len(fdn) > 0 {
		idx := strings.IndexByte(fdn, '.')
		fn := fdn
		if idx >= 0 {
			fn = fdn[:idx]
		}
		fd, err := curr.md.ColumnByStoreName(fn)
		if err != nil {
			return nil, err
		}
		// если тут еще нет поля - заполним
		if curr.fd == nil {
			curr.fd = fd
		} else {
			// ищем, может уже есть такое поле на этом уровне
			for n := curr; n.next != nil; n = n.next {
				curr = n
				if n.fd == fd {
					break
				}
			}
			if curr.fd != fd {
				next := &nodeField{
					fd:     fd,
					md:     curr.md,
					parent: curr,
				}
				curr.next = next
				curr = next
			}
		}
		if idx < 0 {
			break
		}
		fdn = fdn[idx+1:]
		// если поле является моделью - ищем или создаем потомка
		if childMd, err := MDbyType(fd.StructField.Type); err == nil {

			child := &nodeField{
				md:     childMd,
				parent: curr,
			}
			curr.child = child
			curr = child
		}
	}
	return nil
}

func (s *SelectView[T]) Fields(structFieldNames ...string) *SelectView[T] {
	if s.Err != nil {
		return s
	}
	if s.md == nil {
		val := *(new(T))
		md, err := (MD[T]{Val: val}).MD()
		if err != nil {
			s.Err = err
			return s
		}
		s.md = md
	}
	for _, fn := range structFieldNames {
		fd, err := s.md.ColumnByStoreName(fn)
		if err != nil {
			s.Err = err
			return s
		}

	}
}
