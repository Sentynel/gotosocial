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

package transport

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
)

func (t *transport) Finger(ctx context.Context, targetUsername string, targetDomain string) ([]byte, error) {
	l := t.log.WithField("func", "Finger")
	urlString := fmt.Sprintf("https://%s/.well-known/webfinger?resource=acct:%s@%s", targetDomain, targetUsername, targetDomain)
	l.Debugf("performing GET to %s", urlString)

	iri, err := url.Parse(urlString)
	if err != nil {
		return nil, fmt.Errorf("Finger: error parsing url %s: %s", urlString, err)
	}

	l.Debugf("performing GET to %s", iri.String())

	req, err := http.NewRequestWithContext(ctx, "GET", iri.String(), nil)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Accept", "application/json")
	req.Header.Add("Accept", "application/jrd+json")
	req.Header.Add("Date", t.clock.Now().UTC().Format("Mon, 02 Jan 2006 15:04:05")+" GMT")
	req.Header.Add("User-Agent", fmt.Sprintf("%s %s", t.appAgent, t.gofedAgent))
	req.Header.Set("Host", iri.Host)
	t.getSignerMu.Lock()
	err = t.getSigner.SignRequest(t.privkey, t.pubKeyID, req, nil)
	t.getSignerMu.Unlock()
	if err != nil {
		return nil, err
	}
	resp, err := t.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET request to %s failed (%d): %s", iri.String(), resp.StatusCode, resp.Status)
	}
	return ioutil.ReadAll(resp.Body)
}