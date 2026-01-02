package se

type userRow struct {
	ID              int64  `xml:"Id,attr"`
	DisplayName     string `xml:"DisplayName,attr"`
	Reputation      int64  `xml:"Reputation,attr"`
	CreationDate    string `xml:"CreationDate,attr"`
	LastAccessDate  string `xml:"LastAccessDate,attr"`
	WebsiteURL      string `xml:"WebsiteUrl,attr"`
	Location        string `xml:"Location,attr"`
	AboutMe         string `xml:"AboutMe,attr"`
	ProfileImageURL string `xml:"ProfileImageUrl,attr"`
}

type tagRow struct {
	ID            int64  `xml:"Id,attr"`
	TagName       string `xml:"TagName,attr"`
	Count         int64  `xml:"Count,attr"`
	ExcerptPostID int64  `xml:"ExcerptPostId,attr"`
	WikiPostID    int64  `xml:"WikiPostId,attr"`
	CreationDate  string `xml:"CreationDate,attr"`
	Excerpt       string `xml:"Excerpt,attr"`
	Wiki          string `xml:"Wiki,attr"`
}

type postRow struct {
	ID               int64  `xml:"Id,attr"`
	PostTypeID       int64  `xml:"PostTypeId,attr"`
	ParentID         int64  `xml:"ParentId,attr"`
	AcceptedAnswerID int64  `xml:"AcceptedAnswerId,attr"`
	CreationDate     string `xml:"CreationDate,attr"`
	LastEditDate     string `xml:"LastEditDate,attr"`
	OwnerUserID      int64  `xml:"OwnerUserId,attr"`
	Title            string `xml:"Title,attr"`
	Body             string `xml:"Body,attr"`
	Tags             string `xml:"Tags,attr"`
	Score            int64  `xml:"Score,attr"`
	ViewCount        int64  `xml:"ViewCount,attr"`
	AnswerCount      int64  `xml:"AnswerCount,attr"`
	CommentCount     int64  `xml:"CommentCount,attr"`
	FavoriteCount    int64  `xml:"FavoriteCount,attr"`
	ClosedReason     string `xml:"ClosedReason,attr"`
}

type commentRow struct {
	ID           int64  `xml:"Id,attr"`
	PostID       int64  `xml:"PostId,attr"`
	Score        int64  `xml:"Score,attr"`
	Text         string `xml:"Text,attr"`
	CreationDate string `xml:"CreationDate,attr"`
	UserID       int64  `xml:"UserId,attr"`
}

type voteRow struct {
	ID           int64  `xml:"Id,attr"`
	PostID       int64  `xml:"PostId,attr"`
	VoteTypeID   int64  `xml:"VoteTypeId,attr"`
	CreationDate string `xml:"CreationDate,attr"`
	UserID       int64  `xml:"UserId,attr"`
}

type badgeRow struct {
	ID     int64  `xml:"Id,attr"`
	UserID int64  `xml:"UserId,attr"`
	Name   string `xml:"Name,attr"`
	Date   string `xml:"Date,attr"`
	Class  int64  `xml:"Class,attr"`
}
