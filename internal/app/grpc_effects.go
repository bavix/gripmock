package app

import (
	"context"

	"github.com/cockroachdb/errors"
	"github.com/goccy/go-json"
	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/bavix/gripmock/v3/internal/infra/stuber"
	"github.com/bavix/gripmock/v3/internal/infra/template"
)

func (m *grpcMocker) applyEffects(
	ctx context.Context,
	matched *stuber.Stub,
	templateData template.Data,
) {
	if len(matched.Effects) == 0 {
		return
	}

	prepared := make([]effectOperation, 0, len(matched.Effects))

	for i, effect := range matched.Effects {
		op, err := m.prepareEffect(effect, templateData, matched.Session)
		if err != nil {
			zerolog.Ctx(ctx).Err(err).
				Str("stub_id", matched.ID.String()).
				Int("effect_index", i).
				Str("effect_action", effect.Action).
				Msg("failed to prepare effect; skip all effects for request")

			return
		}

		prepared = append(prepared, op)
	}

	for i, op := range prepared {
		if err := m.applyEffectOperation(op); err != nil {
			zerolog.Ctx(ctx).Err(err).
				Str("stub_id", matched.ID.String()).
				Int("effect_index", i).
				Str("effect_action", op.action).
				Msg("failed to apply prepared effect")
		}
	}
}

type effectOperation struct {
	action        string
	upsertStub    *stuber.Stub
	deleteID      uuid.UUID
	parentSession string
}

func (m *grpcMocker) prepareEffect(
	effect stuber.Effect,
	templateData template.Data,
	parentSession string,
) (effectOperation, error) {
	switch effect.Action {
	case stuber.EffectActionUpsert:
		upsert, err := m.prepareUpsertEffect(effect, templateData, parentSession)
		if err != nil {
			return effectOperation{}, err
		}

		return effectOperation{action: effect.Action, upsertStub: upsert, parentSession: parentSession}, nil
	case stuber.EffectActionDelete:
		deleteID, err := m.prepareDeleteEffect(effect, templateData)
		if err != nil {
			return effectOperation{}, err
		}

		return effectOperation{action: effect.Action, deleteID: deleteID, parentSession: parentSession}, nil
	default:
		return effectOperation{}, errors.New("unknown effect action")
	}
}

func (m *grpcMocker) prepareUpsertEffect(
	effect stuber.Effect,
	templateData template.Data,
	parentSession string,
) (*stuber.Stub, error) {
	if len(effect.Stub) == 0 {
		return nil, errors.New("upsert effect requires stub payload")
	}

	payload := deepCopyMapAny(effect.Stub)
	if err := m.templateEngine.ProcessMap(payload, templateData); err != nil {
		return nil, errors.Wrap(err, "failed to process effect upsert templates")
	}

	stub, err := decodeEffectStub(payload)
	if err != nil {
		return nil, err
	}

	if stub.ID == uuid.Nil {
		stub.ID = uuid.New()
	}

	stub.Session = parentSession
	stub.Source = stuber.SourceRest

	if err := m.validator.Struct(stub); err != nil {
		return nil, errors.Wrap(err, "invalid generated upsert effect stub")
	}

	return stub, nil
}

func (m *grpcMocker) prepareDeleteEffect(
	effect stuber.Effect,
	templateData template.Data,
) (uuid.UUID, error) {
	idString := effect.ID
	if idString == "" {
		return uuid.Nil, errors.New("delete effect requires id")
	}

	if template.IsTemplateString(idString) {
		renderedID, err := m.templateEngine.Render(idString, templateData)
		if err != nil {
			return uuid.Nil, errors.Wrap(err, "failed to process effect delete id template")
		}

		idString = renderedID
	}

	id, err := uuid.Parse(idString)
	if err != nil {
		return uuid.Nil, errors.Wrap(err, "invalid effect delete id")
	}

	return id, nil
}

func (m *grpcMocker) applyEffectOperation(op effectOperation) error {
	switch op.action {
	case stuber.EffectActionUpsert:
		if op.upsertStub == nil {
			return errors.New("prepared upsert effect has nil stub")
		}

		m.budgerigar.PutMany(op.upsertStub)

		return nil
	case stuber.EffectActionDelete:
		existing := m.budgerigar.FindByID(op.deleteID)
		if existing == nil || !effectCanDeleteStub(existing, op.parentSession) {
			return nil
		}

		m.budgerigar.DeleteByID(op.deleteID)

		return nil
	default:
		return errors.New("unknown prepared effect action")
	}
}

func effectCanDeleteStub(stub *stuber.Stub, targetSession string) bool {
	if targetSession == "" {
		return stub.Session == ""
	}

	return stub.Session == targetSession
}

func decodeEffectStub(payload map[string]any) (*stuber.Stub, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal effect stub payload")
	}

	generated := &stuber.Stub{}
	if err := json.Unmarshal(body, generated); err != nil {
		return nil, errors.Wrap(err, "failed to decode effect stub payload")
	}

	return generated, nil
}
