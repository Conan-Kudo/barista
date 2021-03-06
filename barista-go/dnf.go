package barista

import (
	"fmt"
	"strings"
	"time"

	"github.com/Necroforger/dgwidgets"
	"github.com/bwmarrin/discordgo"
	"github.com/dustin/go-humanize"
	"github.com/godbus/dbus"
)

type Package struct {
	name        string
	desc        string
	vers        string
	downsize    int
	installsize int
}

type Distro struct {
	matches      []string
	displayName  string
	queryKitName string
	iconURL      string
	colour       int
}

var Distros []Distro = []Distro{
	{
		displayName:  "openSUSE Tumbleweed",
		queryKitName: "tumbleweed",
		matches:      []string{"opensuse", "os", "tumbleweed", "tw"},
		iconURL:      "https://en.opensuse.org/images/c/cd/Button-colour.png",
		colour:       0x73ba25,
	},
	{
		displayName:  "openSUSE Leap",
		queryKitName: "leap",
		matches:      []string{"leap", "opensuse-leap", "os-leap"},
		iconURL:      "https://en.opensuse.org/images/c/cd/Button-colour.png",
		colour:       0x73ba25,
	},
	{
		displayName:  "Fedora",
		queryKitName: "fedora",
		matches:      []string{"fedora"},
		iconURL:      "https://fedoraproject.org/w/uploads/archive/e/e5/20110717032101%21Fedora_infinity.png",
		colour:       0x0b57a4,
	},
	{
		displayName:  "Mageia",
		queryKitName: "mageia",
		matches:      []string{"mageia"},
		iconURL:      "https://pbs.twimg.com/profile_images/553311070215892992/lf8QV6oJ_400x400.png",
		colour:       0x2397d4,
	},
	{
		displayName:  "OpenMandriva",
		queryKitName: "openmandriva",
		matches:      []string{"openmandriva"},
		iconURL:      "https://pbs.twimg.com/profile_images/1140547712208822272/dG9610ZK_400x400.jpg",
		colour:       0x40a5da,
	},
}

func resolveDistro(name string) (Distro, bool) {
	var distro Distro
	set := false

	for _, dist := range Distros {
		for _, match := range dist.matches {
			if strings.ToLower(name) == strings.ToLower(match) {
				distro = dist
				set = true
			}
		}
	}

	return distro, set
}

func DnfRepoQuery(s *discordgo.Session, cmd *LexedCommand) {
	helpmsg := "```dsconfig\n" + repoqueryhelp + "\n```"

	cmd.PaginatorPageName = "Package"
	var dist string
	if cmd.GetFlagPair("-d", "--distro") == "" {
		set := getSetting("dnf", "defaultDistro")
		dist = set.getValue(cmd)
	} else {
		dist = cmd.GetFlagPair("-d", "--distro")
	}
	distro, set := resolveDistro(dist)
	if !set {
		embed := NewEmbed().
			SetColor(0xff0000).
			SetTitle("Please specify a distro from the following list: `fedora`, `opensuse`, `mageia`, and `openmandriva`.")
		msgSend := discordgo.MessageSend{
			Embed: embed.MessageEmbed,
		}
		cmd.SendMessage(&msgSend)
		return
	}
	if cmd.GetFlagPair("-f", "--file") == "" &&
		cmd.GetFlagPair("", "--whatconflicts") == "" &&
		cmd.GetFlagPair("", "--whatobsoletes") == "" &&
		cmd.GetFlagPair("", "--whatprovides") == "" &&
		cmd.GetFlagPair("", "--whatrecommends") == "" &&
		cmd.GetFlagPair("", "--whatenhances") == "" &&
		cmd.GetFlagPair("", "--whatsupplements") == "" &&
		cmd.GetFlagPair("", "--whatsuggests") == "" &&
		cmd.GetFlagPair("", "--whatrequires") == "" &&
		cmd.GetFlagPair("", "--provides") == "" &&
		cmd.GetFlagPair("", "--requires") == "" &&
		cmd.GetFlagPair("", "--recommends") == "" &&
		cmd.GetFlagPair("", "--suggests") == "" &&
		cmd.GetFlagPair("", "--supplements") == "" &&
		cmd.GetFlagPair("", "--enhances") == "" &&
		cmd.GetFlagPair("", "--conflicts") == "" &&
		cmd.GetFlagPair("", "--obsoletes") == "" &&
		cmd.GetFlagPair("-l", "--list") != "nil" {
		embed := NewEmbed().
			SetColor(0xff0000).
			SetDescription(helpmsg).
			SetTitle("Please specify a query in your command.")
		msgSend := discordgo.MessageSend{
			Embed: embed.MessageEmbed,
		}
		cmd.SendMessage(&msgSend)
		return
	}

	conn, err := dbus.SessionBus()
	if err != nil {
		embed := NewEmbed().
			SetColor(0xff0000).
			SetTitle("There was an issue connecting to the package querying daemon.")
		msgSend := discordgo.MessageSend{
			Embed: embed.MessageEmbed,
		}
		cmd.SendMessage(&msgSend)
		return
	}
	obj := conn.Object("com.github.Appadeia.QueryKit", "/com/github/Appadeia/QueryKit")

	if cmd.GetFlagPair("", "--provides") != "" ||
		cmd.GetFlagPair("", "--requires") != "" ||
		cmd.GetFlagPair("", "--recommends") != "" ||
		cmd.GetFlagPair("", "--suggests") != "" ||
		cmd.GetFlagPair("", "--supplements") != "" ||
		cmd.GetFlagPair("", "--enhances") != "" ||
		cmd.GetFlagPair("", "--conflicts") != "" ||
		cmd.GetFlagPair("", "--obsoletes") != "" {

		queryKitType := ""

		if cmd.GetFlagPair("", "--provides") != "" {
			queryKitType = "provides"
		}
		if cmd.GetFlagPair("", "--requires") != "" {
			queryKitType = "requires"
		}
		if cmd.GetFlagPair("", "--recommends") != "" {
			queryKitType = "recommends"
		}
		if cmd.GetFlagPair("", "--suggests") != "" {
			queryKitType = "suggests"
		}
		if cmd.GetFlagPair("", "--supplements") != "" {
			queryKitType = "supplements"
		}
		if cmd.GetFlagPair("", "--enhances") != "" {
			queryKitType = "enhances"
		}
		if cmd.GetFlagPair("", "--conflicts") != "" {
			queryKitType = "conflicts"
		}
		if cmd.GetFlagPair("", "--obsoletes") != "" {
			queryKitType = "obsoletes"
		}

		var reldeps []string
		completed := make(chan int)
		go func() {
			for {
				select {
				case <-completed:
					return
				default:
					s.ChannelTyping(cmd.CommandMessage.ChannelID)
				}
				time.Sleep(500)
			}
		}()
		err = obj.Call("com.github.Appadeia.QueryKit.QueryRepoPackage", 0, cmd.Query.Content, queryKitType, distro.queryKitName).Store(&reldeps)
		completed <- 0
		if err != nil {
			embed := NewEmbed().
				SetColor(0xff0000).
				SetTitle("There was an issue querying packages.").
				SetDescription(err.Error())
			msgSend := discordgo.MessageSend{
				Embed: embed.MessageEmbed,
			}
			cmd.SendMessage(&msgSend)
			return
		}
		if len(reldeps) < 2000 {
			embed := NewEmbed().
				SetColor(distro.colour).
				SetTitle(fmt.Sprintf("Query %s for %s", queryKitType, cmd.Query.Content)).
				SetDescription("```"+strings.Join(reldeps, "\n")+"```").
				SetAuthor(fmt.Sprintf("%s Repoquery", distro.displayName), distro.iconURL)

			msgSend := discordgo.MessageSend{
				Embed: embed.MessageEmbed,
			}
			cmd.SendMessage(&msgSend)
			return
		} else {
			cmd.PaginatorPageName = "Page"
			chunkSize := (len(reldeps) + 1999) / 2000
			paginator := dgwidgets.NewPaginator(cmd.Session, cmd.CommandMessage.ChannelID)
			for i := 0; i < len(reldeps); i += chunkSize {
				end := i + chunkSize

				if end > len(reldeps) {
					end = len(reldeps)
				}

				embed := NewEmbed().
					SetColor(distro.colour).
					SetTitle(fmt.Sprintf("Query %s for %s", queryKitType, cmd.Query.Content)).
					SetDescription("```"+strings.Join(reldeps[i:end], "\n")+"```").
					SetAuthor(fmt.Sprintf("%s Repoquery", distro.displayName), distro.iconURL)

				paginator.Add(embed.MessageEmbed)
			}
			cmd.SendPaginator(paginator)
			return
		}
	}
	if cmd.GetFlagPair("-l", "--list") == "nil" {
		var files []string
		completed := make(chan int)
		go func() {
			for {
				select {
				case <-completed:
					return
				default:
					s.ChannelTyping(cmd.CommandMessage.ChannelID)
				}
				time.Sleep(500)
			}
		}()
		err = obj.Call("com.github.Appadeia.QueryKit.ListFiles", 0, cmd.Query.Content, distro.queryKitName).Store(&files)
		completed <- 0
		if err != nil {
			embed := NewEmbed().
				SetColor(0xff0000).
				SetTitle("There was an issue querying packages.").
				SetDescription(err.Error())
			msgSend := discordgo.MessageSend{
				Embed: embed.MessageEmbed,
			}
			cmd.SendMessage(&msgSend)
			return
		}
		if len(files) < 2000 {
			embed := NewEmbed().
				SetColor(distro.colour).
				SetTitle(fmt.Sprintf("Files of %s", cmd.Query.Content)).
				SetDescription("```"+strings.Join(files, "\n")+"```").
				SetAuthor(fmt.Sprintf("%s Repoquery", distro.displayName), distro.iconURL)

			msgSend := discordgo.MessageSend{
				Embed: embed.MessageEmbed,
			}
			cmd.SendMessage(&msgSend)
			return
		} else {
			cmd.PaginatorPageName = "Page"
			chunkSize := (len(files) + 1999) / 2000
			paginator := dgwidgets.NewPaginator(cmd.Session, cmd.CommandMessage.ChannelID)
			for i := 0; i < len(files); i += chunkSize {
				end := i + chunkSize

				if end > len(files) {
					end = len(files)
				}

				embed := NewEmbed().
					SetColor(distro.colour).
					SetTitle(fmt.Sprintf("Files of %s", cmd.Query.Content)).
					SetDescription("```"+strings.Join(files[i:end], "\n")+"```").
					SetAuthor(fmt.Sprintf("%s Repoquery", distro.displayName), distro.iconURL)

				paginator.Add(embed.MessageEmbed)
			}
			cmd.SendPaginator(paginator)
			return
		}

	}

	m := make(map[string]string)
	if val := cmd.GetFlagPair("-f", "--file"); val != "" {
		m["file"] = val
	}
	if val := cmd.GetFlagPair("", "--whatconflicts"); val != "" {
		m["whatconflicts"] = val
	}
	if val := cmd.GetFlagPair("", "--whatobsoletes"); val != "" {
		m["whatobsoletes"] = val
	}
	if val := cmd.GetFlagPair("", "--whatprovides"); val != "" {
		m["whatprovides"] = val
	}
	if val := cmd.GetFlagPair("", "--whatrecommends"); val != "" {
		m["whatrecommends"] = val
	}
	if val := cmd.GetFlagPair("", "--whatenhances"); val != "" {
		m["whatenhances"] = val
	}
	if val := cmd.GetFlagPair("", "--whatsupplements"); val != "" {
		m["whatsupplements"] = val
	}
	if val := cmd.GetFlagPair("", "--whatsuggests"); val != "" {
		m["whatsuggests"] = val
	}
	if val := cmd.GetFlagPair("", "--whatrequires"); val != "" {
		m["whatrequires"] = val
	}

	var pkgs [][]interface{}
	s.ChannelTyping(cmd.CommandMessage.ChannelID)
	completed := make(chan int)
	go func() {
		for {
			select {
			case <-completed:
				return
			default:
				s.ChannelTyping(cmd.CommandMessage.ChannelID)
			}
			time.Sleep(500)
		}
	}()
	err = obj.Call("com.github.Appadeia.QueryKit.QueryRepo", 0, m, distro.queryKitName).Store(&pkgs)
	completed <- 0
	if err != nil {
		embed := NewEmbed().
			SetColor(0xff0000).
			SetTitle("There was an issue querying packages.").
			SetDescription(err.Error())
		msgSend := discordgo.MessageSend{
			Embed: embed.MessageEmbed,
		}
		cmd.SendMessage(&msgSend)
		return
	}
	if len(pkgs) == 0 {
		embed := NewEmbed().
			SetColor(0xff0000).
			SetTitle("No packages were found.")

		msgSend := discordgo.MessageSend{
			Embed: embed.MessageEmbed,
		}
		cmd.SendMessage(&msgSend)
		return
	}
	if cmd.GetFlagPair("-n", "--no-details") != "" {
		pkgnames := []string{}
		for _, pk := range pkgs {
			pkgnames = append(pkgnames, pk[0].(string))
		}
		if len(pkgs) < 2000 {
			embed := NewEmbed().
				SetColor(distro.colour).
				SetTitle("Packages matching your query").
				SetDescription("```"+strings.Join(pkgnames, "\n")+"```").
				SetAuthor(fmt.Sprintf("%s Repoquery", distro.displayName), distro.iconURL)

			msgSend := discordgo.MessageSend{
				Embed: embed.MessageEmbed,
			}
			cmd.SendMessage(&msgSend)
			return
		} else {
			cmd.PaginatorPageName = "Page"
			chunkSize := (len(pkgnames) + 1999) / 2000
			paginator := dgwidgets.NewPaginator(cmd.Session, cmd.CommandMessage.ChannelID)
			for i := 0; i < len(pkgnames); i += chunkSize {
				end := i + chunkSize

				if end > len(pkgnames) {
					end = len(pkgnames)
				}

				embed := NewEmbed().
					SetColor(distro.colour).
					SetTitle("Packages matching your query").
					SetDescription("```"+strings.Join(pkgnames[i:end], "\n")+"```").
					SetAuthor(fmt.Sprintf("%s Repoquery", distro.displayName), distro.iconURL)

				paginator.Add(embed.MessageEmbed)
			}
			cmd.SendPaginator(paginator)
			return
		}
	}
	paginator := dgwidgets.NewPaginator(cmd.Session, cmd.CommandMessage.ChannelID)
	for _, pkg := range pkgs {
		embed := NewEmbed().
			SetColor(distro.colour).
			SetTitle(pkg[0].(string)).
			SetDescription(pkg[1].(string)).
			AddField("Version", pkg[2].(string), true).
			AddField("Download Size", humanize.Bytes(uint64(pkg[3].(int32))), true).
			AddField("Install Size", humanize.Bytes(uint64(pkg[4].(int32))), true).
			SetAuthor(fmt.Sprintf("%s Repoquery", distro.displayName), distro.iconURL).
			SetURL(pkg[5].(string))

		paginator.Add(embed.MessageEmbed)
	}
	cmd.SendPaginator(paginator)
}

func Dnf(s *discordgo.Session, cmd *LexedCommand) {
	cmd.PaginatorPageName = "Package"
	var dist string
	if cmd.GetFlagPair("-d", "--distro") == "" {
		set := getSetting("dnf", "defaultDistro")
		dist = set.getValue(cmd)
	} else {
		dist = cmd.GetFlagPair("-d", "--distro")
	}
	if cmd.Query.Content == "" {
		embed := NewEmbed().
			SetColor(0xff0000).
			SetTitle("Please specify a search term.")
		msgSend := discordgo.MessageSend{
			Embed: embed.MessageEmbed,
		}
		cmd.SendMessage(&msgSend)
		return
	}
	distro, set := resolveDistro(dist)
	if !set {
		embed := NewEmbed().
			SetColor(0xff0000).
			SetTitle("Please specify a distro from the following list: `fedora`, `opensuse`, `mageia`, and `openmandriva`.")
		msgSend := discordgo.MessageSend{
			Embed: embed.MessageEmbed,
		}
		cmd.SendMessage(&msgSend)
		return
	}
	conn, err := dbus.SessionBus()
	if err != nil {
		embed := NewEmbed().
			SetColor(0xff0000).
			SetTitle("There was an issue connecting to the package querying daemon.")
		msgSend := discordgo.MessageSend{
			Embed: embed.MessageEmbed,
		}
		cmd.SendMessage(&msgSend)
		return
	}
	var pkgs [][]interface{}
	obj := conn.Object("com.github.Appadeia.QueryKit", "/com/github/Appadeia/QueryKit")
	s.ChannelTyping(cmd.CommandMessage.ChannelID)
	completed := make(chan int)
	go func() {
		for {
			select {
			case <-completed:
				return
			default:
				s.ChannelTyping(cmd.CommandMessage.ChannelID)
			}
			time.Sleep(500)
		}
	}()
	err = obj.Call("com.github.Appadeia.QueryKit.SearchPackages", 0, cmd.Query.Content, distro.queryKitName).Store(&pkgs)
	completed <- 0
	if err != nil {
		embed := NewEmbed().
			SetColor(0xff0000).
			SetTitle("There was an issue querying packages.").
			SetDescription(err.Error())
		msgSend := discordgo.MessageSend{
			Embed: embed.MessageEmbed,
		}
		cmd.SendMessage(&msgSend)
		return
	}
	paginator := dgwidgets.NewPaginator(cmd.Session, cmd.CommandMessage.ChannelID)
	for _, pkg := range pkgs {
		embed := NewEmbed().
			SetColor(distro.colour).
			SetTitle(pkg[0].(string)).
			SetDescription(pkg[1].(string)).
			AddField("Version", pkg[2].(string), true).
			AddField("Download Size", humanize.Bytes(uint64(pkg[3].(int32))), true).
			AddField("Install Size", humanize.Bytes(uint64(pkg[4].(int32))), true).
			SetAuthor(fmt.Sprintf("%s Package Search", distro.displayName), distro.iconURL).
			SetURL(pkg[5].(string))

		paginator.Add(embed.MessageEmbed)
	}
	if len(pkgs) == 0 {
		embed := NewEmbed().
			SetColor(0xff0000).
			SetTitle(fmt.Sprintf("No packages matching `%s` found", cmd.Query.Content))

		paginator.Add(embed.MessageEmbed)
	}
	cmd.SendPaginator(paginator)
}
