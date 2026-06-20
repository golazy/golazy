package lazyapp

import "strings"

func (m *metadataFiles) renderRobots() ([]byte, error) {
	var out strings.Builder
	for index, rule := range robotRules(m.robots.Rules) {
		if index > 0 {
			out.WriteString("\n")
		}
		writeRobotRule(&out, rule)
	}
	writeRobotSitemaps(&out, m.robots.Sitemaps, m.sitemap)
	writeRobotExtra(&out, m.robots.Extra)
	return []byte(out.String()), nil
}

func robotRules(rules []RobotsRule) []RobotsRule {
	if len(rules) > 0 {
		return rules
	}
	return []RobotsRule{{UserAgent: "*", Allow: []string{"/"}}}
}

func writeRobotRule(out *strings.Builder, rule RobotsRule) {
	userAgent := strings.TrimSpace(rule.UserAgent)
	if userAgent == "" {
		userAgent = "*"
	}
	out.WriteString("User-agent: ")
	out.WriteString(userAgent)
	out.WriteString("\n")
	writeRobotPaths(out, "Allow", rule.Allow)
	writeRobotPaths(out, "Disallow", rule.Disallow)
	if delay := strings.TrimSpace(rule.CrawlDelay); delay != "" {
		out.WriteString("Crawl-delay: ")
		out.WriteString(delay)
		out.WriteString("\n")
	}
}

func writeRobotPaths(out *strings.Builder, directive string, paths []string) {
	for _, path := range paths {
		path = strings.TrimSpace(path)
		if path == "" {
			continue
		}
		out.WriteString(directive)
		out.WriteString(": ")
		out.WriteString(path)
		out.WriteString("\n")
	}
}

func writeRobotSitemaps(out *strings.Builder, sitemaps []string, sitemap SitemapConfig) {
	sitemaps = append([]string(nil), sitemaps...)
	if len(sitemaps) == 0 && !sitemap.Disabled {
		sitemaps = append(sitemaps, "/sitemap.xml")
	}
	for _, value := range sitemaps {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		out.WriteString("\nSitemap: ")
		out.WriteString(absoluteURL(sitemap.BaseURL, value))
		out.WriteString("\n")
	}
}

func writeRobotExtra(out *strings.Builder, lines []string) {
	for _, line := range lines {
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			continue
		}
		out.WriteString(line)
		out.WriteString("\n")
	}
}
