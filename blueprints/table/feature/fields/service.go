package fields

import (
	"context"

	"github.com/go-mizu/blueprints/table/pkg/ulid"
)

// Service implements the fields API.
type Service struct {
	store Store
}

// NewService creates a new fields service.
func NewService(store Store) *Service {
	return &Service{store: store}
}

// Create creates a new field.
func (s *Service) Create(ctx context.Context, userID string, in CreateIn) (*Field, error) {
	field := &Field{
		ID:          ulid.New(),
		TableID:     in.TableID,
		Name:        in.Name,
		Type:        in.Type,
		Description: in.Description,
		Options:     in.Options,
		Width:       200,
		CreatedBy:   userID,
	}

	// Validate type
	if !isValidType(field.Type) {
		return nil, ErrInvalidType
	}

	if err := s.store.Create(ctx, field); err != nil {
		return nil, err
	}

	return field, nil
}

// GetByID retrieves a field by ID.
func (s *Service) GetByID(ctx context.Context, id string) (*Field, error) {
	return s.store.GetByID(ctx, id)
}

// Update updates a field.
func (s *Service) Update(ctx context.Context, id string, in UpdateIn) (*Field, error) {
	field, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if in.Name != nil {
		field.Name = *in.Name
	}
	if in.Description != nil {
		field.Description = *in.Description
	}
	if in.Options != nil {
		field.Options = *in.Options
	}
	if in.Width != nil {
		field.Width = *in.Width
	}
	if in.IsHidden != nil {
		field.IsHidden = *in.IsHidden
	}

	if err := s.store.Update(ctx, field); err != nil {
		return nil, err
	}

	return field, nil
}

// Delete deletes a field.
func (s *Service) Delete(ctx context.Context, id string) error {
	return s.store.Delete(ctx, id)
}

// ListByTable lists all fields in a table.
func (s *Service) ListByTable(ctx context.Context, tableID string) ([]*Field, error) {
	return s.store.ListByTable(ctx, tableID)
}

// Reorder reorders fields.
func (s *Service) Reorder(ctx context.Context, tableID string, fieldIDs []string) error {
	return s.store.Reorder(ctx, tableID, fieldIDs)
}

// AddSelectChoice adds a choice to a select field.
func (s *Service) AddSelectChoice(ctx context.Context, fieldID string, choice *SelectChoice) error {
	if choice.ID == "" {
		choice.ID = ulid.New()
	}
	choice.FieldID = fieldID
	return s.store.AddSelectChoice(ctx, choice)
}

// UpdateSelectChoice updates a select choice.
func (s *Service) UpdateSelectChoice(ctx context.Context, fieldID, choiceID string, in UpdateChoiceIn) error {
	return s.store.UpdateSelectChoice(ctx, choiceID, in)
}

// DeleteSelectChoice deletes a select choice.
func (s *Service) DeleteSelectChoice(ctx context.Context, fieldID, choiceID string) error {
	return s.store.DeleteSelectChoice(ctx, choiceID)
}

// ListSelectChoices lists all choices for a select field.
func (s *Service) ListSelectChoices(ctx context.Context, fieldID string) ([]*SelectChoice, error) {
	return s.store.ListSelectChoices(ctx, fieldID)
}

func isValidType(t string) bool {
	switch t {
	case TypeSingleLineText, TypeLongText, TypeRichText, TypeNumber, TypeCurrency, TypePercent,
		TypeDuration, TypeRating, TypeSingleSelect, TypeMultiSelect, TypeCheckbox,
		TypeDate, TypeDateTime, TypeCreatedTime, TypeLastModifiedTime,
		TypeLink, TypeLookup, TypeRollup, TypeCount,
		TypeCollaborator, TypeCollaborators, TypeCreatedBy, TypeLastModifiedBy,
		TypeAttachment, TypeBarcode, TypeAutoNumber, TypeFormula, TypeButton,
		TypeEmail, TypeURL, TypePhone:
		return true
	// Also accept frontend naming conventions
	case "text", "user", "datetime", "autonumber":
		return true
	}
	return false
}
