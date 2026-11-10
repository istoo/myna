package gui

// programName is the name owl gives GTK on Linux (g_set_prgname, through
// Wails' LinuxOptions.ProgramName). GTK3 makes it the window's Wayland
// app_id and its X11 WM_CLASS ("owl", "Owl"); without it both are the
// executable's name, owl-linux-amd64 when the release is run as
// downloaded, which no desktop entry names, so the dock drew a gear (#2626).
// The entry registerScheme writes names it in StartupWMClass. WebKitGTK keys
// its data folder by it too (~/.local/share/owl), see keepWebData.
const programName = "owl"
