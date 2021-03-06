package cmd

import (
	"github.com/mikaelmello/pingo/core"
	"github.com/spf13/cobra"
)

var (
	settings *core.Settings
)

var rootCmd = &cobra.Command{
	Use:   "pingo [hostname or ip address]",
	Short: "pingo, adding Go to your ping",
	Long:  "pingo is a Go implementation of the ping utility",
	Args:  cobra.ExactArgs(1),
	PreRun: func(cmd *cobra.Command, args []string) {
		if cmd.Flags().Changed("ttl") {
			settings.IsTTLDefault = false
		}
		if cmd.Flags().Changed("count") {
			settings.IsMaxCountDefault = false
		}
		if cmd.Flags().Changed("deadline") {
			settings.IsDeadlineDefault = false
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		r, err := newRunner(args[0], settings)
		if err != nil {
			println(err.Error())
			return
		}

		r.Start()
		err = r.Wait()
		if err != nil {
			println(err.Error())
		}
	},
}

func init() {
	settings = core.DefaultSettings()

	rootCmd.Flags().IntVarP(&settings.TTL, "ttl", "t", settings.TTL, "Set the IP Time to Live.")
	rootCmd.Flags().IntVarP(&settings.MaxCount, "count", "c", settings.MaxCount,
		"Stop after sending count ECHO_REQUEST packets. With deadline option, ping waits for count ECHO_REPLY packets, until the timeout expires.")
	rootCmd.Flags().Float64VarP(&settings.Interval, "interval", "i", settings.Interval,
		"Wait interval seconds between sending each packet. The default is to wait for one second between each packet normally.")
	rootCmd.Flags().IntVarP(&settings.Timeout, "timeout", "W", settings.Timeout,
		"Time to wait for a response, in seconds. The option affects only timeout in absence of any responses, otherwise ping waits for two RTTs.")
	rootCmd.Flags().IntVarP(&settings.Deadline, "deadline", "w", settings.Deadline,
		"Specify a timeout, in seconds, before ping exits regardless of how many packets have been sent or received. In this case ping does not stop after count packet are sent, it waits either for deadline expire or until count probes are answered or for some error notification from network.")
	rootCmd.Flags().BoolVarP(&settings.Flood, "flood", "f", settings.Flood,
		"Flood ping. For every ECHO_REQUEST sent a period '.' is printed, while for ever "+
			"ECHO_REPLY received a backspace is printed. This provides a rapid display of how many "+
			"packets are being dropped. It sets interval to 0.01s between packets. Only available in "+
			"privileged mode.")
	rootCmd.Flags().BoolVarP(&settings.IsPrivileged, "privileged", "p", settings.IsPrivileged,
		"Whether to use privileged mode. If yes, privileged raw ICMP endpoints are used, non-privileged datagram-oriented otherwise. On Linux, to run unprivileged you must enable the setting 'sudo sysctl -w net.ipv4.ping_group_range=\"0   2147483647\"'. In order to run as a privileged user, you can either run as sudo or execute 'setcap cap_net_raw=+ep <bin path>' to the path of the binary. On Windows, you must run as privileged.")
	rootCmd.Flags().Uint32Var(&settings.LoggingLevel, "log-level", settings.LoggingLevel, "Logging level, goes from top priority 0 (Panic) to lowest priority 6 (Trace). Values out of this range log everything.")
}

// Execute executes the root command of the application.
func Execute() error {
	return rootCmd.Execute()
}
