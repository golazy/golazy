package lazyseo

// PageKind is a common page/content kind that maps to Open Graph and JSON-LD.
type PageKind struct {
	OpenGraph string
	Schema    string
}

var (
	WebPage      = PageKind{OpenGraph: "website", Schema: "WebPage"}
	WebSite      = PageKind{OpenGraph: "website", Schema: "WebSite"}
	Article      = PageKind{OpenGraph: "article", Schema: "Article"}
	BlogPosting  = PageKind{OpenGraph: "article", Schema: "BlogPosting"}
	NewsArticle  = PageKind{OpenGraph: "article", Schema: "NewsArticle"}
	Product      = PageKind{OpenGraph: "product", Schema: "Product"}
	Profile      = PageKind{OpenGraph: "profile", Schema: "ProfilePage"}
	Book         = PageKind{OpenGraph: "book", Schema: "Book"}
	Video        = PageKind{OpenGraph: "video.other", Schema: "VideoObject"}
	MusicSong    = PageKind{OpenGraph: "music.song", Schema: "MusicRecording"}
	MusicAlbum   = PageKind{OpenGraph: "music.album", Schema: "MusicAlbum"}
	Place        = PageKind{OpenGraph: "place", Schema: "Place"}
	Restaurant   = PageKind{OpenGraph: "place", Schema: "Restaurant"}
	Organization = PageKind{OpenGraph: "website", Schema: "Organization"}
)
