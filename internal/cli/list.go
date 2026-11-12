package cli

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/aimuzov/happ-cli/internal/config"
	"github.com/aimuzov/happ-cli/internal/xray"
)

func newListCmd(deps *Deps) *cobra.Command {
	var subName string
	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls", "servers"},
		Short:   "List servers in a subscription",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			sub, err := resolveSub(deps.Store, subName)
			if err != nil {
				return err
			}
			servers := sub.Servers()
			if len(servers) == 0 {
				fmt.Printf("Subscription %q has no servers.\n", sub.Name)
				return nil
			}
			tw := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
			fmt.Fprintf(tw, "#\tPROTOCOL\tADDRESS\tTAG\n")
			for i, s := range servers {
				note := s.Protocol
				if !xray.Supported(s.Protocol) {
					note += " (unsupported)"
				}
				fmt.Fprintf(tw, "%d\t%s\t%s:%d\t%s\n", i+1, note, s.Address, s.Port, s.Tag)
			}
			return tw.Flush()
		},
	}
	cmd.Flags().StringVar(&subName, "sub", "", "subscription name (default: active)")
	_ = cmd.RegisterFlagCompletionFunc("sub", completeSubFlag(deps))
	return cmd
}

func newConfigCmd(deps *Deps) *cobra.Command {
	var subName string
	var socksPort, httpPort int
	var excludes, includes []string
	var excludeFile, includeFile, defaultOutbound string
	var dnsProxy bool
	dir, _ := storeDir()
	settings, _ := config.Load(dir)
	if settings == nil {
		settings = config.Default()
	}
	excludes, includes = settings.Connect.Exclude, settings.Connect.Include
	excludeFile, includeFile, defaultOutbound = settings.Connect.ExcludeFile, settings.Connect.IncludeFile, settings.Connect.DefaultOutbound
	dnsProxy = settings.Connect.DNSProxy
	cmd := &cobra.Command{
		Use:               "config [selector]",
		Short:             "Print the generated xray-core config for a server",
		Args:              cobra.MaximumNArgs(1),
		ValidArgsFunction: completeServerSelector(deps),
		RunE: func(cmd *cobra.Command, args []string) error {
			sub, err := resolveSub(deps.Store, subName)
			if err != nil {
				return err
			}
			selector := ""
			if len(args) > 0 {
				selector = args[0]
			}
			srv, _, err := selectServer(sub.Servers(), selector)
			if err != nil {
				return err
			}
			dir, err := storeDir()
			if err != nil {
				return err
			}
			excludeEntries, err := loadExcludeEntries(excludes, excludeFile, dir)
			if err != nil {
				return err
			}
			includeEntries, err := loadIncludeEntries(includes, includeFile, dir)
			if err != nil {
				return err
			}
			cfg, err := xray.BuildConfig(srv, xray.Options{SocksPort: socksPort, HTTPPort: httpPort, DNSProxy: dnsProxy, Exclude: excludeEntries, Include: includeEntries, DefaultOutbound: defaultOutbound})
			if err != nil {
				return err
			}
			raw, err := cfg.JSON()
			if err != nil {
				return err
			}
			_, err = fmt.Fprintln(cmd.OutOrStdout(), string(raw))
			return err
		},
	}
	cmd.Flags().StringVar(&subName, "sub", "", "subscription name (default: active)")
	cmd.Flags().IntVar(&socksPort, "socks", settings.Connect.SocksPort, "SOCKS5 inbound port")
	cmd.Flags().IntVar(&httpPort, "http", settings.Connect.HTTPPort, "HTTP inbound proxy port")
	cmd.Flags().StringArrayVar(&excludes, "exclude", excludes, "domain or IPv4 CIDR to send direct (repeatable)")
	cmd.Flags().StringVar(&excludeFile, "exclude-file", excludeFile, "file with exclusion entries")
	cmd.Flags().StringArrayVar(&includes, "include", includes, "domain or IPv4 CIDR to send through proxy (repeatable)")
	cmd.Flags().StringVar(&includeFile, "include-file", includeFile, "file with included entries")
	cmd.Flags().StringVar(&defaultOutbound, "default-outbound", defaultOutbound, "default route: proxy or direct")
	cmd.Flags().BoolVar(&dnsProxy, "dns-proxy", dnsProxy, "route DNS through proxy")
	_ = cmd.RegisterFlagCompletionFunc("sub", completeSubFlag(deps))
	return cmd
}
