package node

import (
	"github.com/urfave/cli"
	"DNA/cli/common"
	"fmt"
	"os"
)

func NewCommon()*cli.Command{
	return &cli.Command{
		Name :"node",
		Usage : "node manage",
		Description:"",
		ArgsUsage:"[args]",
		Flags:[]cli.Flag{
			cli.BoolFlag{
				Name:"export,e",
				Usage:"export all blocks in db",
			},
			cli.IntFlag{
				Name:"to",
				Usage:"the last block to export. Zero means current block height",
			},
		},
		Action:nodeAction,
		OnUsageError:onUsageError,
	}
}

func nodeAction(ctx *cli.Context)error{
	export := ctx.Bool("export")
	to := ctx.Int("to")
	if export{
		err := ExportBlocks("", to)
		if err != nil {
			fmt.Fprintf(os.Stdout, "exportBlocks error %s\n", err)
		}
	}

	return nil
}

func onUsageError(ctx *cli.Context, err error, isSubCmd bool)error{
	common.PrintError(ctx, err, "node")
	return cli.NewExitError("", 1)
}
