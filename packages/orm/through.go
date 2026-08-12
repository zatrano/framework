package orm

// HasManyThrough loads related models through an intermediate model.
// throughForeignKey is the column on Through referencing Parent (e.g. parent_id).
// relatedForeignKey is the column on Through referencing Related (e.g. role_id).
func HasManyThrough[Parent, Through, Related any](
	parent *Parent,
	throughForeignKey, relatedForeignKey string,
	localKey ...string,
) ([]Related, error) {
	local := defaultLocalKey[Parent](localKey...)
	parentKey, err := attribute(parent, local)
	if err != nil {
		return nil, err
	}
	if parentKey == nil {
		return []Related{}, nil
	}

	throughRows, err := Where[Through](throughForeignKey, parentKey).Get()
	if err != nil {
		return nil, err
	}
	if len(throughRows) == 0 {
		return []Related{}, nil
	}

	relatedKey := KeyName[Related]()
	ids := make([]any, 0, len(throughRows))
	for i := range throughRows {
		id, err := attribute(&throughRows[i], relatedForeignKey)
		if err != nil {
			return nil, err
		}
		if id != nil {
			ids = append(ids, id)
		}
	}
	if len(ids) == 0 {
		return []Related{}, nil
	}
	return Query[Related]().WhereIn(relatedKey, ids).Get()
}

// HasOneThrough returns the first related model through an intermediate model.
func HasOneThrough[Parent, Through, Related any](
	parent *Parent,
	throughForeignKey, relatedForeignKey string,
	localKey ...string,
) (*Related, error) {
	items, err := HasManyThrough[Parent, Through, Related](parent, throughForeignKey, relatedForeignKey, localKey...)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, nil
	}
	return &items[0], nil
}
