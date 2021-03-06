package ui

import (
    "strings"
    "os/exec"
    "strconv"
    "runtime"

    "github.com/gizak/termui"

    "../tracker"
    "../file"
)

type UI struct {
    current_page int
    tracker_text *termui.Par
    key *termui.Par
    files_list *termui.List
    gauge *termui.Gauge
    trackers []*tracker.Tracker
    files []*file.File
    file_chan chan int

    selecting_file bool
    file_selected bool
    selected_file int

    first_file int
    last_file int
}

func NewUI() *UI {
    ui := &UI{}
    ui.first_file = 0
    ui.last_file = 6

    return ui
}

func (u *UI) update_trackers_text() {
    text := ""
    for _, t := range u.trackers {
        if t.IsConnected() {
            text = text + "  [Tracker :: " + t.GetUrl() + "](fg-cyan)\n"
        } else {
            text = text + "  [Tracker :: " + t.GetUrl() + "](fg-red)\n"
        }
    }

    u.tracker_text.Text = text
}

func (u *UI) Init(name string, trackers []*tracker.Tracker) {
    u.trackers = trackers

    err := termui.Init()
    if err != nil {
        panic(err)
    }
    defer termui.Close()

    uvg_welcome_text := []string{
        "                                                                                                  ",
        "[▄• ▄▌ ▌ ▐· ▄▄ • ▄▄▄▄▄      ▄▄▄  ▄▄▄  ▄▄▄ . ▐ ▄ ▄▄▄▄▄     ▄▄▄·▄▄▄  ▄▄▄ ..▄▄ · ▄▄▄ . ▐ ▄ ▄▄▄▄▄.▄▄ · ](fg-cyan)",
        "[█▪██▌▪█·█▌▐█ ▀ ▪•██  ▪     ▀▄ █·▀▄ █·▀▄.▀·•█▌▐█•██      ▐█ ▄█▀▄ █·▀▄.▀·▐█ ▀. ▀▄.▀·•█▌▐█•██  ▐█ ▀. ](fg-cyan)",
        "[█▌▐█▌▐█▐█•▄█ ▀█▄ ▐█.▪ ▄█▀▄ ▐▀▀▄ ▐▀▀▄ ▐▀▀▪▄▐█▐▐▌ ▐█.▪     ██▀·▐▀▀▄ ▐▀▀▪▄▄▀▀▀█▄▐▀▀▪▄▐█▐▐▌ ▐█.▪▄▀▀▀█▄](fg-cyan)",
        "[▐█▄█▌ ███ ▐█▄▪▐█ ▐█▌·▐█▌.▐▌▐█•█▌▐█•█▌▐█▄▄▌██▐█▌ ▐█▌·    ▐█▪·•▐█•█▌▐█▄▄▌▐█▄▪▐█▐█▄▄▌██▐█▌ ▐█▌·▐█▄▪▐█](fg-cyan)",
        "[ ▀▀▀ . ▀  ·▀▀▀▀  ▀▀▀  ▀█▄▀▪.▀  ▀.▀  ▀ ▀▀▀ ▀▀ █▪ ▀▀▀     .▀   .▀  ▀ ▀▀▀  ▀▀▀▀  ▀▀▀ ▀▀ █▪ ▀▀▀  ▀▀▀▀ ](fg-cyan)",
        "[                                                                                                  ](fg-cyan)",
        "██████████████████████████████████████████████████████████████████████████████████████████████████",
        "                                                                                                  "}

    for _, r := range uvg_welcome_text {
        p := termui.NewPar(r)
        p.TextFgColor = termui.ColorRed
        p.Border = false
        p.Height = 1

        termui.Body.AddRows(termui.NewRow(termui.NewCol(2, 0, nil), termui.NewCol(11, 0, p)))
    }
    
    u.tracker_text = termui.NewPar("");
    u.tracker_text.Height = len(u.trackers) + 2
    u.tracker_text.Width = 1
    u.tracker_text.BorderLabel = "Torrent :: " + name
    u.tracker_text.BorderLabelFg = termui.ColorCyan
    u.update_trackers_text();
    termui.Body.AddRows(termui.NewRow(termui.NewCol(2, 0, nil), termui.NewCol(8, 0, u.tracker_text)))

    strs := []string{}

    u.files_list = termui.NewList()
    u.files_list.Items = strs
    u.files_list.BorderLabelFg = termui.ColorCyan
    u.files_list.BorderLabel = "Files "
    u.files_list.Height = 3 + u.last_file
    u.files_list.Width = 25
    u.files_list.Y = 0

    termui.Body.AddRows(termui.NewRow(termui.NewCol(2, 0, nil), termui.NewCol(8, 0, u.files_list)))

    u.gauge = termui.NewGauge()
    u.gauge.Percent = 0
    u.gauge.Height = 3
    u.gauge.BarColor = termui.ColorCyan
    u.gauge.PercentColorHighlighted = termui.ColorBlack
    u.gauge.Label = "Loading..."
    termui.Body.AddRows(termui.NewRow(termui.NewCol(2, 0, nil), termui.NewCol(8, 0, u.gauge)))

    u.key = termui.NewPar("  [up    -> file list up](fg-red) \n  [down  -> file list down](fg-red) \n  [enter -> start download](fg-red) \n  [v     -> open video in vlc](fg-red) \n  [q     -> quit](fg-cyan)");
    u.key.Height = len(u.trackers) + 2
    u.key.Width = 1
    termui.Body.AddRows(termui.NewRow(termui.NewCol(2, 0, nil), termui.NewCol(8, 0, u.key)))

    // calculate layout
    termui.Body.Align()

    termui.Render(termui.Body)

    termui.Handle("/sys/kbd/<up>", func(termui.Event) {
        if u.selecting_file == true {
            if u.selected_file > 0 {
                u.selected_file--
                if u.selected_file < u.first_file {
                    u.first_file--
                    u.last_file--
                }
            }

            u.update_files_text()
            u.Refresh()
        }
    })

    termui.Handle("/sys/kbd/<down>", func(termui.Event) {
        if u.selecting_file == true {
            if u.selected_file < len(u.files) {
                u.selected_file++
                if u.selected_file > u.last_file {
                    u.first_file++
                    u.last_file++
                }
            }

            u.update_files_text()
            u.Refresh()
        }
    })

    termui.Handle("/sys/kbd/<enter>", func(termui.Event) {
        // up arrow
        if u.selecting_file == true {
            u.file_selected = true
            u.selecting_file = false

            u.key.Text = "  [up    -> file list up](fg-red) \n  [down  -> file list down](fg-red) \n  [enter -> start download](fg-red) \n  [v     -> open video in vlc](fg-red) \n  [q     -> quit](fg-cyan)"

            u.file_chan <- u.selected_file
        }
    })

    termui.Handle("/sys/kbd/v", func(termui.Event) {
        // launch vlc
        if u.file_selected == true {
            f := u.files[u.selected_file]

            if runtime.GOOS == "windows" && runtime.GOARCH == "amd64" {
                path := strings.Join(f.GetPath(), "\\")
                cmd := exec.Command("START", "\"C:\\Program Files (x86)\\VideoLAN\\VLC\\vlc.exe\"", path)
                cmd.Run()
            } else {
                path := strings.Join(f.GetPath(), "/")
                cmd := exec.Command("vlc", path, "&")
                cmd.Run()
            }
        }
    })

    termui.Handle("/sys/kbd/q", func(termui.Event) {
        // enter
        termui.StopLoop()
    })

    termui.Handle("/sys/wnd/resize", func(e termui.Event) {
        termui.Body.Width = termui.TermWidth()
        u.Refresh()
    })

    termui.Handle("/timer/1s", func(e termui.Event) {
        u.update_trackers_text()
        u.Refresh()
    })

    termui.Loop()
}

func (u *UI) update_files_text() {
    strs := []string{}
    
    for i, f := range u.files {
        if i >= u.first_file && i <= u.last_file {
            path := strings.Join(f.GetDisplayPath(), "/")
            if i == u.selected_file {
                strs = append(strs, "[[::] " + path + "](fg-cyan)")
            } else {
                strs = append(strs, "[[  ] " + path + "](fg-red)")
            }
        }
    }
    if u.selected_file > len(u.files) - 6 {
        if u.selected_file == len(u.files) {
            strs = append(strs, "[[::] all](fg-cyan)")
        } else {
            strs = append(strs, "[[  ] all](fg-red)")
        }
    }

    u.files_list.Items = strs
}

func (u *UI) SelectFile(files []*file.File, file_chan chan int) {
    u.gauge.Label = "Selecting file to view..."
    u.key.Text = "  [up    -> file list up](fg-cyan) \n  [down  -> file list down](fg-cyan) \n  [enter -> start download](fg-cyan) \n  [v     -> open video in vlc](fg-red) \n  [q     -> quit](fg-cyan)"

    u.file_chan = file_chan
    u.files = files
    u.update_files_text()

    u.selecting_file = true

    u.Refresh()
}

func (u *UI) Refresh() {
    termui.Body.Width = termui.TermWidth()
    termui.Body.Align()
    termui.Clear()
    termui.Render(termui.Body)
}

func (u *UI) SetPercent(completed int, total int) {
    var f float64 = float64(completed) / float64(total) * 100
    u.gauge.Percent = int(f)
    u.gauge.Label = "{{percent}}% (" + strconv.FormatInt(int64(completed), 10) + " / " + strconv.FormatInt(int64(total), 10) + " chunks completed)"
    if u.gauge.Percent >= 10 {
        u.key.Text = "  [up    -> file list up](fg-red) \n  [down  -> file list down](fg-red) \n  [enter -> start download](fg-red) \n  [v     -> open video in vlc](fg-cyan) \n  [q     -> quit](fg-cyan)"
    }
}