package pan

import (
	"context"
	"net/http"
)

func (client *Client) Rename(ctx context.Context, token, id, name string) error {
	return client.mutateFile(ctx, token, "/open/ufile/update", map[string]string{"file_id": id, "file_name": name})
}

func (client *Client) Move(ctx context.Context, token, id, parent string) error {
	return client.mutateFile(ctx, token, "/open/ufile/move", map[string]string{"file_ids": id, "to_cid": parent})
}

// Delete sends an already verified empty directory to the recycle bin.
func (client *Client) Delete(ctx context.Context, token, id string) error {
	return client.mutateFile(ctx, token, "/open/ufile/delete", map[string]string{"file_ids": id})
}

func (client *Client) mutateFile(ctx context.Context, token, endpoint string, form map[string]string) error {
	_, err := apiRequest[apiResponse](client,
		client.http.R().SetContext(ctx).SetAuthToken(token).SetFormData(form),
		http.MethodPost, apiURL+endpoint, "file organization")
	return err
}
