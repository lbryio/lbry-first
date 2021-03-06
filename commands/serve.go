package commands

import (
	"github.com/lbryio/lbry-first/commands/server"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	serveCmd.PersistentFlags().StringP("host", "", "0.0.0.0", "host to listen on")
	serveCmd.PersistentFlags().IntP("port", "p", 1100, "port binding used for the rpc server")
	//Bind to Viper
	err := viper.BindPFlags(serveCmd.PersistentFlags())
	if err != nil {
		logrus.Panic(err)
	}
	rootCmd.AddCommand(serveCmd)
}

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Runs the LBRY First JSON RPC server",
	Long:  `Runs the LBRY First JSON RPC server`,
	Args:  cobra.OnlyValidArgs,
	Run: func(cmd *cobra.Command, args []string) {
		server.Start()
	},
}
