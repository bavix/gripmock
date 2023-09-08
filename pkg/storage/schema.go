package storage

import (
	"github.com/hashicorp/go-memdb"
)

const (
	TableName          = "stubs"
	IDField            = "id"
	ServiceField       = "service"
	ServiceMethodField = "service_method"
)

func schema() map[string]*memdb.TableSchema {
	return map[string]*memdb.TableSchema{
		TableName: {
			Name: TableName,
			Indexes: map[string]*memdb.IndexSchema{
				IDField: {
					Name:    IDField,
					Unique:  true,
					Indexer: &UUIDFieldIndex{Field: "ID"},
				},
				ServiceField: {
					Name:    ServiceField,
					Unique:  false,
					Indexer: &memdb.StringFieldIndex{Field: "Service"},
				},
				ServiceMethodField: {
					Name:   ServiceMethodField,
					Unique: false,
					Indexer: &memdb.CompoundMultiIndex{
						Indexes: []memdb.Indexer{
							&memdb.StringFieldIndex{Field: "Service"},
							&memdb.StringFieldIndex{Field: "Method"},
						},
						AllowMissing: false,
					},
				},
			},
		},
	}
}
