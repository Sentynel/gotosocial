/*
   GoToSocial
   Copyright (C) 2021 GoToSocial Authors admin@gotosocial.org

   This program is free software: you can redistribute it and/or modify
   it under the terms of the GNU Affero General Public License as published by
   the Free Software Foundation, either version 3 of the License, or
   (at your option) any later version.

   This program is distributed in the hope that it will be useful,
   but WITHOUT ANY WARRANTY; without even the implied warranty of
   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
   GNU Affero General Public License for more details.

   You should have received a copy of the GNU Affero General Public License
   along with this program.  If not, see <http://www.gnu.org/licenses/>.
*/

package status

import (
	"context"
	"fmt"
	"time"

	apimodel "github.com/superseriousbusiness/gotosocial/internal/api/model"
	"github.com/superseriousbusiness/gotosocial/internal/gtserror"
	"github.com/superseriousbusiness/gotosocial/internal/gtsmodel"
	"github.com/superseriousbusiness/gotosocial/internal/id"
	"github.com/superseriousbusiness/gotosocial/internal/text"
	"github.com/superseriousbusiness/gotosocial/internal/util"
)

func (p *processor) Create(ctx context.Context, account *gtsmodel.Account, application *gtsmodel.Application, form *apimodel.AdvancedStatusCreateForm) (*apimodel.Status, gtserror.WithCode) {
	uris := util.GenerateURIsForAccount(account.Username, p.config.Protocol, p.config.Host)
	thisStatusID, err := id.NewULID()
	if err != nil {
		return nil, gtserror.NewErrorInternalError(err)
	}
	thisStatusURI := fmt.Sprintf("%s/%s", uris.StatusesURI, thisStatusID)
	thisStatusURL := fmt.Sprintf("%s/%s", uris.StatusesURL, thisStatusID)

	newStatus := &gtsmodel.Status{
		ID:                       thisStatusID,
		URI:                      thisStatusURI,
		URL:                      thisStatusURL,
		CreatedAt:                time.Now(),
		UpdatedAt:                time.Now(),
		Local:                    true,
		AccountID:                account.ID,
		AccountURI:               account.URI,
		ContentWarning:           text.RemoveHTML(form.SpoilerText),
		ActivityStreamsType:      gtsmodel.ActivityStreamsNote,
		Sensitive:                form.Sensitive,
		Language:                 form.Language,
		CreatedWithApplicationID: application.ID,
		Text:                     form.Status,
	}

	if err := p.ProcessReplyToID(ctx, form, account.ID, newStatus); err != nil {
		return nil, gtserror.NewErrorInternalError(err)
	}

	if err := p.ProcessMediaIDs(ctx, form, account.ID, newStatus); err != nil {
		return nil, gtserror.NewErrorInternalError(err)
	}

	if err := p.ProcessVisibility(ctx, form, account.Privacy, newStatus); err != nil {
		return nil, gtserror.NewErrorInternalError(err)
	}

	if err := p.ProcessLanguage(ctx, form, account.Language, newStatus); err != nil {
		return nil, gtserror.NewErrorInternalError(err)
	}

	if err := p.ProcessMentions(ctx, form, account.ID, newStatus); err != nil {
		return nil, gtserror.NewErrorInternalError(err)
	}

	if err := p.ProcessTags(ctx, form, account.ID, newStatus); err != nil {
		return nil, gtserror.NewErrorInternalError(err)
	}

	if err := p.ProcessEmojis(ctx, form, account.ID, newStatus); err != nil {
		return nil, gtserror.NewErrorInternalError(err)
	}

	if err := p.ProcessContent(ctx, form, account.ID, newStatus); err != nil {
		return nil, gtserror.NewErrorInternalError(err)
	}

	// put the new status in the database
	if err := p.db.PutStatus(ctx, newStatus); err != nil {
		return nil, gtserror.NewErrorInternalError(err)
	}

	// send it back to the processor for async processing
	p.fromClientAPI <- gtsmodel.FromClientAPI{
		APObjectType:   gtsmodel.ActivityStreamsNote,
		APActivityType: gtsmodel.ActivityStreamsCreate,
		GTSModel:       newStatus,
		OriginAccount:  account,
	}

	// return the frontend representation of the new status to the submitter
	mastoStatus, err := p.tc.StatusToMasto(ctx, newStatus, account)
	if err != nil {
		return nil, gtserror.NewErrorInternalError(fmt.Errorf("error converting status %s to frontend representation: %s", newStatus.ID, err))
	}

	return mastoStatus, nil
}