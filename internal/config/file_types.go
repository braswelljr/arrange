package config

const defaultUnknownExtFolderName = "Other"

var defaultKnownFiles = []FileExt{
	{
		Folder:      "Pictures",
		ExemptFiles: false,
		Extensions: []string{
			"jpg", "jpeg", "png", "gif", "webp", "cr2",
			"tif", "tiff", "bmp", "heif", "heic", "avif",
			"jxr", "psd", "ico", "dwg", "svg", "eps",
			"ai", "arw", "nef", "dng", "raw",
		},
	},
	{
		Folder:      "Videos",
		ExemptFiles: false,
		Extensions: []string{
			"mp4", "m4v", "mkv", "webm", "mov", "avi",
			"wmv", "mpg", "flv", "3gp", "vob", "ts",
			"m2ts", "mts", "divx", "ogv", "f4v", "rm", "rmvb",
		},
	},
	{
		Folder:      "Applications",
		ExemptFiles: false,
		Extensions: []string{
			"wasm", "dex", "dey", "exe", "dmg", "rpm",
			"deb", "pkg", "apk", "msi", "appimage", "ipa", "run",
		},
	},
	{
		Folder:      "Fonts",
		ExemptFiles: false,
		Extensions:  []string{"woff", "woff2", "ttf", "otf", "fon", "fnt", "pfb", "pfm"},
	},
	{
		Folder:      "Documents",
		ExemptFiles: false,
		Extensions: []string{
			"doc", "docx", "xls", "xlsx", "ppt", "pptx",
			"pdf", "rtf", "txt", "csv", "odt", "ods",
			"odp", "md", "pages", "numbers", "keynote", "tex",
		},
	},
	{
		Folder:      "eBooks",
		ExemptFiles: false,
		Extensions:  []string{"mobi", "azw", "azw3", "epub", "fb2", "djvu"},
	},
	{
		Folder:      "Audio",
		ExemptFiles: false,
		Extensions: []string{
			"mid", "mp3", "m4a", "ogg", "flac", "wav",
			"amr", "aac", "opus", "wma", "aiff", "aif", "ape", "mka", "wv",
		},
	},
	{
		Folder:      "Archive",
		ExemptFiles: false,
		Extensions: []string{
			"zip", "tar", "rar", "gz", "bz2", "7z",
			"xz", "zstd", "zst", "lzma", "tgz", "tbz2", "txz",
			"swf", "eot", "ps", "nes", "crx", "cab",
			"ar", "Z", "lz", "elf", "dcm", "jar", "war",
		},
	},
	{
		Folder:      "DiskImages",
		ExemptFiles: false,
		Extensions:  []string{"img", "iso", "vmdk", "vhd", "vdi", "qcow2"},
	},
	{
		Folder:      "Database",
		ExemptFiles: false,
		Extensions:  []string{"sqlite", "sqlite3", "sql", "db", "mdb", "accdb"},
	},
	{
		Folder:      "Torrents",
		ExemptFiles: false,
		Extensions:  []string{"torrent"},
	},
	{
		Folder:      "Code",
		ExemptFiles: true,
		Extensions: []string{
			"py", "js", "ts", "go", "rs", "cpp", "cc", "c", "h", "hpp",
			"java", "kt", "swift", "rb", "php", "cs", "sh", "bash", "zsh",
			"ps1", "bat", "cmd", "lua", "dart", "vue", "jsx", "tsx",
			"html", "css", "scss", "sass", "less", "r", "scala",
			"ex", "exs", "pl", "tf", "yaml", "yml", "toml", "json", "xml",
		},
	},
	{
		Folder:      "",
		ExemptFiles: true,
		Extensions:  []string{"download", "crdownload", ".DS_Store"},
	},
}
