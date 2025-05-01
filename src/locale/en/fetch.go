package en

// --- Fetch Tool ---

const FetchToolDescription = "Fetches content from a given URL. It can simplify HTML content or return raw data."

const FetchToolArgURLDescription = "The URL to fetch content from."

const FetchToolArgMaxLengthDescription = "Maximum number of characters to return from the content (default: 10000)."

const FetchToolArgStartIndexDescription = "Starting character index for fetching content (default: 0). Useful for pagination."

const FetchToolArgRawDescription = "If true, returns the raw content (e.g., HTML) instead of trying to simplify it (default: false)."

const FetchToolArgIgnoreRobotsTxtDescription = "If true, ignores the site's robots.txt rules (default: false). Use responsibly."

const FetchToolArgUserAgentDescription = "Custom User-Agent string to use for the request. Defaults based on context (autonomous vs manual)."

const FetchToolArgProxyURLDescription = "Optional proxy URL to use for the request (e.g., 'http://user:pass@host:port'). Uses environment proxy if unset."

const FetchToolArgIsManualRequestDescription = "Indicates if the request is triggered manually by a user action (affects default User-Agent and robots.txt check behavior)."
