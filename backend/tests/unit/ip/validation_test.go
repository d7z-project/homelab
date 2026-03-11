package ip_test

import (
	"homelab/pkg/models"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIPModelValidation(t *testing.T) {
	t.Run("IPGroup Validation", func(t *testing.T) {
		tests := []struct {
			name    string
			group   models.IPPool
			wantErr bool
		}{
			{
				name:    "Valid Group",
				group:   models.IPPool{ID: "valid_id", Meta: models.IPPoolV1Meta{Name: "Valid Name"}},
				wantErr: false,
			},
			{
				name:    "Invalid ID - Uppercase",
				group:   models.IPPool{ID: "InvalidID", Meta: models.IPPoolV1Meta{Name: "Name"}},
				wantErr: true,
			},
			{
				name:    "Invalid ID - Special Chars",
				group:   models.IPPool{ID: "id@123", Meta: models.IPPoolV1Meta{Name: "Name"}},
				wantErr: true,
			},
			{
				name:    "Empty Name",
				group:   models.IPPool{ID: "valid_id", Meta: models.IPPoolV1Meta{Name: "   "}},
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
				export: models.IPExport{ID: "need_fix", Meta: models.IPExportV1Meta{
					Name:     "Valid Name",
					Rule:     "true",
					GroupIDs: []string{"g1"}},
				},
				wantErr: false,
			},
			{
				name: "Missing Rule",
				export: models.IPExport{ID: "need_fix", Meta: models.IPExportV1Meta{
					Name:     "Name",
					GroupIDs: []string{"g1"}},
				},
				wantErr: true,
			},
			{
				name: "Missing Groups",
				export: models.IPExport{ID: "need_fix", Meta: models.IPExportV1Meta{
					Name: "Name",
					Rule: "true",
				}},
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
