package sheets

import (
	"context"
	"errors"
	"time"

	"github.com/go-mizu/blueprints/spreadsheet/pkg/ulid"
)

var (
	ErrNotFound = errors.New("sheet not found")
)

// Service implements the sheets API.
type Service struct {
	store Store
}

// NewService creates a new sheets service.
func NewService(store Store) *Service {
	return &Service{store: store}
}

// Create creates a new sheet.
func (s *Service) Create(ctx context.Context, in *CreateIn) (*Sheet, error) {
	now := time.Now()

	// Auto-calculate index if not specified
	index := in.Index
	if index == 0 {
		// Get existing sheets and find max index
		existing, err := s.store.ListByWorkbook(ctx, in.WorkbookID)
		if err == nil && len(existing) > 0 {
			maxIndex := 0
			for _, sh := range existing {
				if sh.Index > maxIndex {
					maxIndex = sh.Index
				}
			}
			index = maxIndex + 1
		}
	}

	sheet := &Sheet{
		ID:               ulid.New(),
		WorkbookID:       in.WorkbookID,
		Name:             in.Name,
		Index:            index,
		Color:            in.Color,
		GridColor:        "#E2E8F0",
		DefaultRowHeight: 21,
		DefaultColWidth:  100,
		RowHeights:       make(map[int]int),
		ColWidths:        make(map[int]int),
		HiddenRows:       []int{},
		HiddenCols:       []int{},
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	if err := s.store.Create(ctx, sheet); err != nil {
		return nil, err
	}

	return sheet, nil
}

// GetByID retrieves a sheet by ID.
func (s *Service) GetByID(ctx context.Context, id string) (*Sheet, error) {
	return s.store.GetByID(ctx, id)
}

// List lists sheets in a workbook.
func (s *Service) List(ctx context.Context, workbookID string) ([]*Sheet, error) {
	return s.store.ListByWorkbook(ctx, workbookID)
}

// Update updates a sheet.
func (s *Service) Update(ctx context.Context, id string, in *UpdateIn) (*Sheet, error) {
	sheet, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if in.Name != "" {
		sheet.Name = in.Name
	}
	if in.Index > 0 {
		sheet.Index = in.Index
	}
	if in.Hidden != nil {
		sheet.Hidden = *in.Hidden
	}
	if in.Color != "" {
		sheet.Color = in.Color
	}
	if in.GridColor != "" {
		sheet.GridColor = in.GridColor
	}
	if in.FrozenRows != nil {
		sheet.FrozenRows = *in.FrozenRows
	}
	if in.FrozenCols != nil {
		sheet.FrozenCols = *in.FrozenCols
	}
	if in.DefaultRowHeight != nil {
		sheet.DefaultRowHeight = *in.DefaultRowHeight
	}
	if in.DefaultColWidth != nil {
		sheet.DefaultColWidth = *in.DefaultColWidth
	}

	sheet.UpdatedAt = time.Now()

	if err := s.store.Update(ctx, sheet); err != nil {
		return nil, err
	}

	return sheet, nil
}

// Delete deletes a sheet.
func (s *Service) Delete(ctx context.Context, id string) error {
	return s.store.Delete(ctx, id)
}

// Copy creates a copy of a sheet.
func (s *Service) Copy(ctx context.Context, id string, newName string) (*Sheet, error) {
	original, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	sheet := &Sheet{
		ID:               ulid.New(),
		WorkbookID:       original.WorkbookID,
		Name:             newName,
		Index:            original.Index + 1,
		Color:            original.Color,
		GridColor:        original.GridColor,
		FrozenRows:       original.FrozenRows,
		FrozenCols:       original.FrozenCols,
		DefaultRowHeight: original.DefaultRowHeight,
		DefaultColWidth:  original.DefaultColWidth,
		RowHeights:       copyIntMap(original.RowHeights),
		ColWidths:        copyIntMap(original.ColWidths),
		HiddenRows:       copyIntSlice(original.HiddenRows),
		HiddenCols:       copyIntSlice(original.HiddenCols),
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	if err := s.store.Create(ctx, sheet); err != nil {
		return nil, err
	}

	return sheet, nil
}

// SetRowHeight sets the height of a specific row.
func (s *Service) SetRowHeight(ctx context.Context, sheetID string, row int, height int) error {
	sheet, err := s.store.GetByID(ctx, sheetID)
	if err != nil {
		return err
	}

	if sheet.RowHeights == nil {
		sheet.RowHeights = make(map[int]int)
	}
	sheet.RowHeights[row] = height
	sheet.UpdatedAt = time.Now()

	return s.store.Update(ctx, sheet)
}

// SetColWidth sets the width of a specific column.
func (s *Service) SetColWidth(ctx context.Context, sheetID string, col int, width int) error {
	sheet, err := s.store.GetByID(ctx, sheetID)
	if err != nil {
		return err
	}

	if sheet.ColWidths == nil {
		sheet.ColWidths = make(map[int]int)
	}
	sheet.ColWidths[col] = width
	sheet.UpdatedAt = time.Now()

	return s.store.Update(ctx, sheet)
}

// HideRow hides a row.
func (s *Service) HideRow(ctx context.Context, sheetID string, row int) error {
	sheet, err := s.store.GetByID(ctx, sheetID)
	if err != nil {
		return err
	}

	// Check if already hidden
	for _, r := range sheet.HiddenRows {
		if r == row {
			return nil
		}
	}

	sheet.HiddenRows = append(sheet.HiddenRows, row)
	sheet.UpdatedAt = time.Now()

	return s.store.Update(ctx, sheet)
}

// HideCol hides a column.
func (s *Service) HideCol(ctx context.Context, sheetID string, col int) error {
	sheet, err := s.store.GetByID(ctx, sheetID)
	if err != nil {
		return err
	}

	// Check if already hidden
	for _, c := range sheet.HiddenCols {
		if c == col {
			return nil
		}
	}

	sheet.HiddenCols = append(sheet.HiddenCols, col)
	sheet.UpdatedAt = time.Now()

	return s.store.Update(ctx, sheet)
}

// ShowRow shows a hidden row.
func (s *Service) ShowRow(ctx context.Context, sheetID string, row int) error {
	sheet, err := s.store.GetByID(ctx, sheetID)
	if err != nil {
		return err
	}

	newHidden := make([]int, 0, len(sheet.HiddenRows))
	for _, r := range sheet.HiddenRows {
		if r != row {
			newHidden = append(newHidden, r)
		}
	}
	sheet.HiddenRows = newHidden
	sheet.UpdatedAt = time.Now()

	return s.store.Update(ctx, sheet)
}

// ShowCol shows a hidden column.
func (s *Service) ShowCol(ctx context.Context, sheetID string, col int) error {
	sheet, err := s.store.GetByID(ctx, sheetID)
	if err != nil {
		return err
	}

	newHidden := make([]int, 0, len(sheet.HiddenCols))
	for _, c := range sheet.HiddenCols {
		if c != col {
			newHidden = append(newHidden, c)
		}
	}
	sheet.HiddenCols = newHidden
	sheet.UpdatedAt = time.Now()

	return s.store.Update(ctx, sheet)
}

func copyIntMap(m map[int]int) map[int]int {
	if m == nil {
		return nil
	}
	result := make(map[int]int, len(m))
	for k, v := range m {
		result[k] = v
	}
	return result
}

func copyIntSlice(s []int) []int {
	if s == nil {
		return nil
	}
	result := make([]int, len(s))
	copy(result, s)
	return result
}
