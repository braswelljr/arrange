package config

// Folder name constants — import these instead of using raw strings so a
// rename here propagates everywhere automatically.
const (
	FolderPictures   = "Pictures"
	FolderVideos     = "Videos"
	FolderAudio      = "Audio"
	FolderDocuments  = "Documents"
	FolderEBooks     = "eBooks"
	FolderApps       = "Applications"
	FolderArchive    = "Archive"
	FolderDiskImages = "DiskImages"
	FolderFonts      = "Fonts"
	FolderDatabase   = "Database"
	FolderSubtitles  = "Subtitles"
	FolderDesign     = "Design"
	FolderModels3D   = "3D Models"
	FolderCode       = "Code"
	FolderOther      = "Other"
)

const defaultUnknownExtFolderName = FolderOther

// defaultExcludedDirs is intentionally empty — files from Telegram, WhatsApp,
// and other app folders are organised normally once they are complete.
// Populate this (or set excluded_dirs in config.json) only to permanently
// block a specific folder.
var defaultExcludedDirs = []string{}

// defaultKnownFiles is the canonical extension → folder table.
// Rules:
//   - ExemptFiles:true  →  file is NEVER moved (in-progress or source code)
//   - First matching entry wins for a given extension
//   - Torrent files and all in-progress download artefacts are exempt so the
//     organiser never touches an unfinished download
var defaultKnownFiles = []FileExt{

	// ── Pictures ────────────────────────────────────────────────────────────
	{
		Folder:      FolderPictures,
		ExemptFiles: false,
		Extensions: []string{
			// JPEG family
			"jpg", "jpeg", "jfif", "jpe",
			// PNG family
			"png", "apng",
			// GIF / animation
			"gif", "gifv",
			// Modern
			"webp", "avif", "jxl", "jxr", "heif", "heic", "bpg",
			// Classic
			"bmp", "ico", "tif", "tiff", "pcx", "tga",
			// HDR / compositing
			"hdr", "exr",
			// Vector
			"svg", "svgz", "eps",
			// Photoshop / GIMP
			"psd", "psb", "xcf",
			// Camera RAW
			"cr2", "cr3", "crw", // Canon
			"arw", "sr2", "srf", // Sony
			"nef", "nrw", // Nikon
			"orf",               // Olympus
			"rw2",               // Panasonic
			"raf",               // Fuji
			"dng",               // Adobe DNG
			"raw", "3fr", "mef", // Hasselblad / generic
			// Misc
			"pbm", "pgm", "ppm", "pnm", "xbm",
		},
	},

	// ── Videos ──────────────────────────────────────────────────────────────
	{
		Folder:      FolderVideos,
		ExemptFiles: false,
		Extensions: []string{
			// Universal containers
			"mp4", "m4v", "mkv", "webm", "mov",
			// AVI / Windows
			"avi", "wmv", "asf", "divx", "xvid",
			// MPEG family
			"mpg", "mpeg", "mpe", "m1v", "m2v",
			// Transport streams (TV / Blu-ray)
			"ts", "m2ts", "mts", "tp", "trp",
			// DVD
			"vob", "ifo",
			// Flash / streaming
			"flv", "f4v",
			// Mobile / 3GPP
			"3gp", "3g2",
			// Ogg / open
			"ogv", "ogm",
			// RealMedia
			"rm", "rmvb",
			// Misc
			"amv", "nsv", "rec", "mxf",
		},
	},

	// ── Audio ────────────────────────────────────────────────────────────────
	{
		Folder:      FolderAudio,
		ExemptFiles: false,
		Extensions: []string{
			// Lossy
			"mp3", "m4a", "aac", "ogg", "oga", "opus", "wma", "amr",
			// Lossless
			"flac", "wav", "aiff", "aif", "ape", "wv", "alac", "tta",
			// Voice notes (Telegram / WhatsApp)
			"m4r", "oga",
			// MIDI / tracker
			"mid", "midi", "mod", "xm", "it", "s3m",
			// Misc
			"mka", "mpc", "ra", "au", "snd", "caf", "gsm", "spx",
		},
	},

	// ── Documents ────────────────────────────────────────────────────────────
	{
		Folder:      FolderDocuments,
		ExemptFiles: false,
		Extensions: []string{
			// Microsoft Office
			"doc", "docx", "docm", "dot", "dotx", "dotm",
			"xls", "xlsx", "xlsm", "xlsb", "xlt", "xltx", "csv", "tsv",
			"ppt", "pptx", "pptm", "pot", "potx",
			// PDF / PostScript
			"pdf", "ps", "xps", "oxps",
			// Open / LibreOffice
			"odt", "ods", "odp", "odg", "odf", "ott", "ots",
			// Apple iWork
			"pages", "numbers", "keynote",
			// Plain text / markup
			"txt", "rtf", "md", "rst", "tex", "log", "nfo",
			// Misc
			"wpd", "wps", "hwp", "pub", "vsd", "vsdx",
		},
	},

	// ── eBooks ───────────────────────────────────────────────────────────────
	{
		Folder:      FolderEBooks,
		ExemptFiles: false,
		Extensions: []string{
			"epub", "mobi", "azw", "azw3", "azw4", "kfx",
			"fb2", "fb3", "djvu", "lit", "lrf", "pdb", "prc",
			// Comic book archives
			"cbr", "cbz", "cb7", "cbt", "cbf",
		},
	},

	// ── Applications / Installers ─────────────────────────────────────────────
	{
		Folder:      FolderApps,
		ExemptFiles: false,
		Extensions: []string{
			// Windows
			"exe", "msi", "msix", "msixbundle", "appx", "appxbundle", "cab",
			// macOS
			"dmg", "pkg", "mpkg",
			// Linux
			"deb", "rpm", "appimage", "flatpak", "snap", "run",
			// Android / iOS
			"apk", "aab", "xapk", "ipa",
			// Cross-platform
			"jar", "war", "wasm", "dex", "dey",
		},
	},

	// ── Archives ─────────────────────────────────────────────────────────────
	{
		Folder:      FolderArchive,
		ExemptFiles: false,
		Extensions: []string{
			// ZIP family
			"zip", "zipx",
			// TAR family
			"tar", "tgz", "tbz2", "txz", "tlz", "tzst",
			// Compressed
			"gz", "bz2", "xz", "zst", "zstd", "lzma", "lz", "lz4", "br",
			// RAR (split: r00, r01, …)
			"rar", "r00", "r01", "r02", "r03",
			// 7-Zip
			"7z",
			// Legacy
			"ar", "Z", "ace", "arc", "arj",
			// Package-like
			"crx", "nupkg",
		},
	},

	// ── Disk Images ──────────────────────────────────────────────────────────
	{
		Folder:      FolderDiskImages,
		ExemptFiles: false,
		Extensions:  []string{"img", "iso", "vmdk", "vhd", "vhdx", "vdi", "qcow2", "ova", "ovf", "toast", "cdr"},
	},

	// ── Fonts ────────────────────────────────────────────────────────────────
	{
		Folder:      FolderFonts,
		ExemptFiles: false,
		Extensions:  []string{"woff", "woff2", "ttf", "otf", "ttc", "fon", "fnt", "pfb", "pfm", "eot"},
	},

	// ── Database ─────────────────────────────────────────────────────────────
	{
		Folder:      FolderDatabase,
		ExemptFiles: false,
		Extensions:  []string{"sqlite", "sqlite3", "db", "db3", "sql", "mdb", "accdb", "accde", "ndf", "mdf", "ldf"},
	},

	// ── Subtitles ────────────────────────────────────────────────────────────
	{
		Folder:      FolderSubtitles,
		ExemptFiles: false,
		Extensions:  []string{"srt", "ass", "ssa", "vtt", "sub", "sbv", "smi", "idx", "sup", "dfxp", "ttml", "lrc"},
	},

	// ── Design files ─────────────────────────────────────────────────────────
	{
		Folder:      FolderDesign,
		ExemptFiles: false,
		Extensions:  []string{"sketch", "fig", "xd", "ai", "indd", "idml", "afdesign", "afpub", "afphoto", "studio"},
	},

	// ── 3D Models ────────────────────────────────────────────────────────────
	{
		Folder:      FolderModels3D,
		ExemptFiles: false,
		Extensions:  []string{"obj", "fbx", "stl", "blend", "3ds", "dae", "gltf", "glb", "max", "c4d", "mb", "ma", "ply", "step", "stp", "iges"},
	},

	// ── Source code (exempt — never moved) ───────────────────────────────────
	{
		Folder:      FolderCode,
		ExemptFiles: true,
		Extensions: []string{
			"py", "pyi", "js", "mjs", "cjs", "ts", "tsx", "jsx",
			"go", "rs", "cpp", "cc", "c", "h", "hpp", "cs",
			"java", "kt", "kts", "swift", "rb", "php",
			"sh", "bash", "zsh", "fish", "ps1", "bat", "cmd",
			"lua", "dart", "vue", "svelte",
			"html", "htm", "css", "scss", "sass", "less",
			"r", "scala", "ex", "exs", "erl", "pl",
			"tf", "hcl", "yaml", "yml", "toml",
			"json", "json5", "jsonc", "xml", "ini", "cfg", "conf", "env",
		},
	},

	// ── Torrent meta-files (exempt — never moved, never tracked) ─────────────
	// The organiser has no role in torrent workflows.  .torrent and .magnet
	// files stay where they are; the completed download (video, audio, …) is
	// handled by its own file-type rule above once it is fully written.
	{
		Folder:      "",
		ExemptFiles: true,
		Extensions:  []string{"torrent", "magnet"},
	},

	// ── In-progress / partial downloads (exempt — NEVER touch these) ─────────
	// Each entry represents a file that is still being written by a browser,
	// download manager, or torrent client.  Moving or renaming them while
	// they are open would corrupt the download.
	{
		Folder:      "",
		ExemptFiles: true,
		Extensions: []string{
			// ── Browsers ─────────────────────────────────────────────────
			"crdownload", // Chrome / Chromium / Edge / Brave
			"part",       // Firefox, yt-dlp, aria2
			"download",   // Safari / macOS
			"opdownload", // Opera
			// ── Torrent clients ───────────────────────────────────────────
			"!qb", // qBittorrent
			"!ut", // µTorrent
			"!bt", // BitTorrent
			// ── Download managers ─────────────────────────────────────────
			"aria2",     // aria2
			"fdm",       // Free Download Manager
			"jdownload", // JDownloader
			"idm",       // Internet Download Manager
			"idt",       // IDM temp
			"ytdl",      // youtube-dl / yt-dlp
			"axel",      // Axel downloader
			"dlpart",    // Generic
			// ── Generic in-progress markers ───────────────────────────────
			"downloading", "partial", "incomplete", "unfinished",
			// ── macOS filesystem metadata ─────────────────────────────────
			"DS_Store",
		},
	},
}
