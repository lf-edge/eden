package cmd

import (
	"fmt"
	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eden/pkg/eden"
	"github.com/lf-edge/eden/pkg/utils"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"os"
)

var (
	postgresTag   string
	postgresPort  int
	postgresDist  string
	postgresForce bool
	postgresRm    bool
)

var postgresCmd = &cobra.Command{
	Use: "postgres",
}

var startPostgresCmd = &cobra.Command{
	Use:   "start",
	Short: "start postgres",
	Long:  `Start postgres.`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		assignCobraToViper(cmd)
		viperLoaded, err := utils.LoadConfigFile(configFile)
		if err != nil {
			return fmt.Errorf("error reading config: %s", err.Error())
		}
		if viperLoaded {
			postgresTag = viper.GetString("postgres.tag")
			postgresPort = viper.GetInt("postgres.port")
			postgresDist = utils.ResolveAbsPath(viper.GetString("postgres.dist"))
			postgresForce = viper.GetBool("postgres.force")
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		command, err := os.Executable()
		if err != nil {
			log.Fatalf("cannot obtain executable path: %s", err)
		}
		log.Infof("Executable path: %s", command)
		if err := eden.StartPostgres(postgresPort, postgresDist, postgresForce, postgresTag); err != nil {
			log.Errorf("cannot start postgres: %s", err)
		} else {
			log.Infof("postgres is running and accessible on port %d", postgresPort)
		}
	},
}

var stopPostgresCmd = &cobra.Command{
	Use:   "stop",
	Short: "stop postgres",
	Long:  `Stop postgres.`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		assignCobraToViper(cmd)
		viperLoaded, err := utils.LoadConfigFile(configFile)
		if err != nil {
			return fmt.Errorf("error reading config: %s", err.Error())
		}
		if viperLoaded {
			postgresRm = viper.GetBool("postgres-rm")
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		if err := eden.StopPostgres(postgresRm); err != nil {
			log.Errorf("cannot stop postgres: %s", err)
		}
	},
}

var statusPostgresCmd = &cobra.Command{
	Use:   "status",
	Short: "status of postgres",
	Long:  `Status of postgres.`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		assignCobraToViper(cmd)
		_, err := utils.LoadConfigFile(configFile)
		if err != nil {
			return fmt.Errorf("error reading config: %s", err.Error())
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		statuspostgres, err := eden.StatusPostgres()
		if err != nil {
			log.Errorf("cannot obtain status of postgres: %s", err)
		} else {
			fmt.Printf("postgres status: %s\n", statuspostgres)
		}
	},
}

func postgresInit() {
	postgresCmd.AddCommand(startPostgresCmd)
	postgresCmd.AddCommand(stopPostgresCmd)
	postgresCmd.AddCommand(statusPostgresCmd)
	startPostgresCmd.Flags().StringVarP(&postgresTag, "postgres-tag", "", defaults.DefaultPostgresTag, "tag of postgres container to pull")
	startPostgresCmd.Flags().StringVarP(&postgresDist, "postgres-dist", "", "", "postgres dist to start (required)")
	startPostgresCmd.Flags().IntVarP(&postgresPort, "postgres-port", "", defaults.DefaultPostgresPort, "postgres port to start")
	startPostgresCmd.Flags().BoolVarP(&postgresForce, "postgres-force", "", false, "postgres force rebuild")
	stopPostgresCmd.Flags().BoolVarP(&postgresRm, "postgres-rm", "", false, "postgres rm on stop")
}
