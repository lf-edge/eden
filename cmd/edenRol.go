package cmd

import (
	"fmt"

	"github.com/lf-edge/eden/pkg/openevec"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func newRolCmd(configName, verbosity *string) *cobra.Command {
	cfg := &openevec.EdenSetupArgs{}
	var rolCmd = &cobra.Command{
		Use:               "rol",
		Short:             `Manage devices in Rack Of Labs`,
		PersistentPreRunE: preRunViperLoadFunction(cfg, configName, verbosity),
	}

	rolCmd.AddCommand(newRolRentCmd(cfg))

	return rolCmd
}

func newRolRentCmd(cfg *openevec.EdenSetupArgs) *cobra.Command {
	var rolProjectID string

	var rolRentCmd = &cobra.Command{
		Use:   "rent",
		Short: "Manage device rents",
		Long:  `Manage device rents`,
	}

	groups := CommandGroups{
		{
			Message: "Basic Commands",
			Commands: []*cobra.Command{
				newCreateRentCmd(&rolProjectID, cfg),
				newCloseRentCmd(&rolProjectID),
				newGetRentCmd(&rolProjectID),
				newGetRentConsoleOutputCmd(&rolProjectID),
			},
		},
	}

	groups.AddTo(rolRentCmd)

	rolRentCmd.PersistentFlags().StringVarP(&rolProjectID, "project-id", "p", "", "project id")
	_ = rolRentCmd.MarkPersistentFlagRequired("project-id")

	return rolRentCmd
}

func newGetRentConsoleOutputCmd(rolProjectID *string) *cobra.Command {
	var rolRentID string

	var getRentConsoleOutputCmd = &cobra.Command{
		Use:   "console-output",
		Short: "Get device console output",
		Long:  `Get device console output from uart`,
		Run: func(cmd *cobra.Command, args []string) {
			output, err := openevec.GetRentConsoleOutput(*rolProjectID, rolRentID)
			if err != nil {
				log.Fatal(err)
			}
			fmt.Println(output)
		},
	}

	getRentConsoleOutputCmd.Flags().StringVarP(&rolRentID, "id", "i", "", "rent id")
	_ = getRentConsoleOutputCmd.MarkFlagRequired("id")

	return getRentConsoleOutputCmd
}

func newCreateRentCmd(rolProjectID *string, cfg *openevec.EdenSetupArgs) *cobra.Command {
	var rolRentName, rolModel, rolManufacturer, rolIPXEUrl string

	var createRentCmd = &cobra.Command{
		Use:   "create",
		Short: "Create a new device rent",
		Long:  `Create a new device rent`,

		Run: func(cmd *cobra.Command, args []string) {
			err := openevec.CreateRent(*rolProjectID, rolRentName, rolModel, rolManufacturer, rolIPXEUrl, cfg)
			if err != nil {
				log.Fatal(err)
			}
		},
	}

	createRentCmd.Flags().StringVarP(&rolRentName, "name", "n", "", "rent name")
	createRentCmd.Flags().StringVar(&rolModel, "model", "", "device model")
	createRentCmd.Flags().StringVarP(&rolManufacturer, "manufacturer", "m", "", "device manufacturer")
	createRentCmd.Flags().StringVarP(&rolIPXEUrl, "ipxe-cfg-url", "i", "", "url to IPXE cfg file")
	_ = createRentCmd.MarkFlagRequired("name")
	_ = createRentCmd.MarkFlagRequired("model")
	_ = createRentCmd.MarkFlagRequired("manufacturer")

	return createRentCmd
}

func newGetRentCmd(rolProjectID *string) *cobra.Command {
	var rolRentID string

	var getRentCmd = &cobra.Command{
		Use:   "get",
		Short: "Get the device rent",
		Long:  `Get the device rent`,
		Run: func(cmd *cobra.Command, args []string) {
			if err := openevec.GetRent(*rolProjectID, rolRentID); err != nil {
				log.Fatal(err)
			}
		},
	}

	getRentCmd.Flags().StringVarP(&rolRentID, "id", "i", "", "rent id")
	_ = getRentCmd.MarkFlagRequired("id")

	return getRentCmd
}

func newCloseRentCmd(rolProjectID *string) *cobra.Command {
	var rolRentID string

	var closeRentCmd = &cobra.Command{
		Use:   "close",
		Short: "Close the device rent",
		Long:  `Close the device rent`,
		Run: func(cmd *cobra.Command, args []string) {
			if err := openevec.CloseRent(*rolProjectID, rolRentID); err != nil {
				log.Fatal(err)
			}
		},
	}

	closeRentCmd.Flags().StringVarP(&rolRentID, "id", "i", "", "rent id")
	_ = closeRentCmd.MarkFlagRequired("id")

	return closeRentCmd
}
