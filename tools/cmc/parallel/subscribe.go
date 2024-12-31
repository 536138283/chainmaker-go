package parallel

import "github.com/spf13/cobra"

func subscribeCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "subscribe",
		Short: "subscribe",
		RunE: func(_ *cobra.Command, _ []string) error {
			statistician := getStatistician()
			initParallel()
			subNodes(statistician)
			return nil
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&pairsString, "pairs", "a", "[{\"key\":\"key\",\"value\":\"counter1\",\"unique\":false}]", "specify pairs")
	flags.StringVarP(&pairsFile, "pairs-file", "A", "", "specify pairs file, if used, set --pairs=\"\"")
	flags.StringVarP(&method, "method", "m", "increase", "specify contract method")
	flags.StringVarP(&abiPath, "abi-path", "", "", "abi file path")
	flags.StringVarP(&statisticalType, "statistical-type", "", "default", "normal statistical type or block based statistical type, input normal or block default:normal ")
	return cmd
}
