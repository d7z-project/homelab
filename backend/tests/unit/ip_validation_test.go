package unit

import (
	"homelab/pkg/models"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIPModelValidation(t *testing.T) {
	t.Run("IPGroup Validation", func(t *testing.T) {
		tests := []struct {
			name    string
			group   models.IPGroup
			wantErr bool
		}{
			{
				name:    "Valid Group",
				group:   models.IPGroup{ID: "valid_id", Name: "Valid Name"},
				wantErr: false,
			},
			{
				name:    "Invalid ID - Uppercase",
				group:   models.IPGroup{ID: "InvalidID", Name: "Name"},
				wantErr: true,
			},
			{
				name:    "Invalid ID - Special Chars",
				group:   models.IPGroup{ID: "id@123", Name: "Name"},
				wantErr: true,
			},
			{
				name:    "Empty Name",
				group:   models.IPGroup{ID: "valid_id", Name: "   "},
				wantErr: true,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				err := tt.group.Bind(nil)
				if tt.wantErr {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
				}
			})
		}
	})

	t.Run("IPExport Validation", func(t *testing.T) {
		tests := []struct {
			name    string
			export  models.IPExport
			wantErr bool
		}{
			{
				name: "Valid Export",
				export: models.IPExport{
					ID:       "valid_export",
					Name:     "Valid Name",
					Rule:     "true",
					GroupIDs: []string{"g1"},
				},
				wantErr: false,
			},
			{
				name: "Missing Rule",
				export: models.IPExport{
					ID:       "valid_export",
					Name:     "Name",
					GroupIDs: []string{"g1"},
				},
				wantErr: true,
			},
			{
				name: "Missing Groups",
				export: models.IPExport{
					ID:   "valid_export",
					Name: "Name",
					Rule: "true",
				},
				wantErr: true,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				err := tt.export.Bind(nil)
				if tt.wantErr {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
				}
			})
		}
	})

	t.Run("IPPoolEntryRequest Validation", func(t *testing.T) {
		req := &models.IPPoolEntryRequest{CIDR: "  ", NewTags: []string{"  Upper  "}}
		err := req.Bind(nil)
		assert.Error(t, err, "Should fail on empty CIDR")

		req = &models.IPPoolEntryRequest{CIDR: "1.1.1.1", OldTags: []string{" OLD "}, NewTags: []string{" NEW "}}
		err = req.Bind(nil)
		assert.NoError(t, err)
		assert.Equal(t, "old", req.OldTags[0])
		assert.Equal(t, "new", req.NewTags[0])
	})
}
