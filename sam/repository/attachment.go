package repository

import (
	"context"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/titpetric/factory"

	"github.com/crusttech/crust/sam/types"
)

type (
	AttachmentRepository interface {
		With(ctx context.Context, db *factory.DB) AttachmentRepository

		FindAttachmentByID(id uint64) (*types.Attachment, error)
		FindAttachmentByMessageID(IDs ...uint64) (types.MessageAttachmentSet, error)
		CreateAttachment(mod *types.Attachment) (*types.Attachment, error)
		DeleteAttachmentByID(id uint64) error
		BindAttachment(attachmentId, messageId uint64) error
	}

	attachment struct {
		*repository
	}
)

const (
	sqlAttachmentColumns = `
		a.id, a.rel_user,
		a.url, a.preview_url,
		a.name,
		a.meta,
		a.created_at, a.updated_at,	a.deleted_at
	`
	sqlAttachmentScope = "deleted_at IS NULL"

	sqlAttachmentByID = `SELECT ` + sqlAttachmentColumns + ` FROM attachments AS a WHERE id = ? AND ` + sqlAttachmentScope

	sqlAttachmentByMessageID = `SELECT ` + sqlAttachmentColumns + `, rel_message
		      FROM attachments AS a
		           INNER JOIN message_attachment AS ma ON a.id = ma.rel_attachment 
		     WHERE ma.rel_message IN (?) AND ` + sqlAttachmentScope

	ErrAttachmentNotFound = repositoryError("AttachmentNotFound")
)

func Attachment(ctx context.Context, db *factory.DB) AttachmentRepository {
	return (&attachment{}).With(ctx, db)
}

func (r *attachment) With(ctx context.Context, db *factory.DB) AttachmentRepository {
	return &attachment{
		repository: r.repository.With(ctx, db),
	}
}

func (r *attachment) FindAttachmentByID(id uint64) (*types.Attachment, error) {
	mod := &types.Attachment{}

	return mod, isFound(r.db().Get(mod, sqlAttachmentByID, id), mod.ID > 0, ErrAttachmentNotFound)
}

func (r *attachment) FindAttachmentByMessageID(IDs ...uint64) (rval types.MessageAttachmentSet, err error) {
	rval = make([]*types.MessageAttachment, 0)

	if len(IDs) == 0 {
		return
	}

	if sql, args, err := sqlx.In(sqlAttachmentByMessageID, IDs); err != nil {
		return nil, err
	} else {
		return rval, r.db().Select(&rval, sql, args...)
	}
}

func (r *attachment) CreateAttachment(mod *types.Attachment) (*types.Attachment, error) {
	if mod.ID == 0 {
		mod.ID = factory.Sonyflake.NextID()
	}

	mod.CreatedAt = time.Now()

	return mod, r.db().Insert("attachments", mod)
}

func (r *attachment) DeleteAttachmentByID(id uint64) error {
	return r.updateColumnByID("attachments", "deleted_at", nil, id)
}

func (r *attachment) BindAttachment(attachmentId, messageId uint64) error {
	bond := struct {
		RelAttachment uint64 `db:"rel_attachment"`
		RelMessage    uint64 `db:"rel_message"`
	}{attachmentId, messageId}

	return r.db().Insert("message_attachment", bond)
}
