package cmd

import (
	"os"
	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/prvst/philosopher/lib/err"
	"github.com/prvst/philosopher/lib/qua"
	"github.com/prvst/philosopher/lib/sys"
	"github.com/spf13/cobra"
)

// labelquantCmd represents the labelquant command
var labelquantCmd = &cobra.Command{
	Use:   "labelquant",
	Short: "Isobaric Labeling-Based Relative Quantification ",
	Run: func(cmd *cobra.Command, args []string) {

		if len(m.UUID) < 1 && len(m.Home) < 1 {
			e := &err.Error{Type: err.WorkspaceNotFound, Class: err.FATA}
			logrus.Fatal(e.Error())
		}

		if len(m.Quantify.Format) < 1 || len(m.Quantify.Dir) < 1 {
			logrus.Fatal("You need to provide the path to the mz files and the correct extension.")
		}

		if len(m.Quantify.Plex) < 1 {
			logrus.Fatal("You need to especify the experiment Plex")
		}

		// hardcoded tmt for now
		logrus.Info("Executing label-based quantification")
		m.Quantify.Brand = "tmt"

		if strings.EqualFold(strings.ToLower(m.Quantify.Format), "mzml") {
			m.Quantify.Format = "mzML"
		} else if strings.EqualFold(m.Quantify.Format, "mzxml") {
			m.Quantify.Format = "mzXML"
		} else {
			logrus.Fatal("Unknown file format")
		}

		// if len(lbl.ChanNorm) > 0 && lbl.IntNorm == true {
		// 	i, err := strconv.Atoi(lbl.ChanNorm)
		// 	if i > 10 || i < 1 || err != nil {
		// 		logrus.Fatal("Inexisting channel number:", lbl.ChanNorm)
		// 	}
		// 	logrus.Fatal("You can choose only one method of normalization")
		// } else if len(lbl.ChanNorm) == 0 && lbl.IntNorm == false {
		// 	logrus.Fatal("Missing normalization method, type 'philosopher labelquant --help' for more information")
		// }

		//err := lbl.RunLabeledQuantification()
		err := qua.RunTMTQuantification(m.Quantify)
		if err != nil {
			logrus.Fatal(err)
		}

		// store paramters on meta data
		m.Serialize()

		logrus.Info("Done")
		return
	},
}

func init() {

	if len(os.Args) > 1 && os.Args[1] == "labelquant" {

		m.Restore(sys.Meta())

		labelquantCmd.Flags().StringVarP(&m.Quantify.Plex, "plex", "", "", "number of channels")
		labelquantCmd.Flags().StringVarP(&m.Quantify.Dir, "dir", "", "", "folder path containing the raw files")
		labelquantCmd.Flags().StringVarP(&m.Quantify.Format, "ext", "", "", "spectra file extension (mzML, mzXML)")
		//labelquantCmd.Flags().StringVarP(&m.Quantify.Annot, "annot", "", "", "annotation file with custom names for the tags")
		labelquantCmd.Flags().Float64VarP(&m.Quantify.Tol, "tol", "", 20, "m/z tolerance in ppm")
		labelquantCmd.Flags().Float64VarP(&m.Quantify.Purity, "purity", "", 0.5, "ion purity threshold")
		labelquantCmd.Flags().BoolVarP(&m.Quantify.Unique, "uniqueonly", "", false, "report quantification based on only unique peptides")
	}

	RootCmd.AddCommand(labelquantCmd)
}
