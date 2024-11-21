package main

type TweetsResponse struct {
	Data struct {
		User struct {
			Result struct {
				Timeline_v2 struct {
					Timeline struct {
						Instructions []struct {
							Type    string
							Entries []struct {
								// Cursors have "cursor-bottom-1234567...", tweets have "tweet-123456..."
								EntryId string `json:"entryId"`
								Content struct {
									Value       string `json:"value,omitempty"`
									ItemContent struct {
										// itemType      string
										Tweet_results struct {
											Result struct {
												RestID string `json:"rest_id"`
												Legacy struct {
													Retweeted_status_result *struct{}
													// user_id_str             string
													// Id                  string `json:"id_str"`
												}
											}
										}
									} `json:"itemContent,omitempty"`
								}
							}
						}
					}
				}
			}
		}
	}
}