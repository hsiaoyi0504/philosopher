package rep

import (
	"encoding/gob"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/prvst/cmsl/err"
	"github.com/prvst/philosopher/lib/sys"
)

// Serialize converts the whle structure to a gob file
func (e *Evidence) Serialize() *err.Error {

	//TODO fix error name convetion

	// create a file
	dataFile, er := os.Create(sys.EvBin())
	if er != nil {
		return &err.Error{Type: err.CannotCreateOutputFile, Class: err.FATA, Argument: er.Error()}
	}

	dataEncoder := gob.NewEncoder(dataFile)
	goberr := dataEncoder.Encode(e)
	if goberr != nil {
		return &err.Error{Type: err.CannotSerializeData, Class: err.FATA, Argument: goberr.Error()}
	}
	dataFile.Close()

	return nil
}

// SerializeGranular converts the whole structure into sevral small gob files
func (e *Evidence) SerializeGranular() *err.Error {

	// create EV PSM
	er := SerializeEVPSM(e)
	if er != nil {
		return er
	}

	// create EV Ion
	er = SerializeEVIon(e)
	if er != nil {
		return er
	}

	// create EV Peptides
	er = SerializeEVPeptides(e)
	if er != nil {
		return er
	}

	// create EV Ion
	er = SerializeEVProteins(e)
	if er != nil {
		return er
	}

	// create EV Mods
	er = SerializeEVMods(e)
	if er != nil {
		return er
	}

	// create EV Modifications
	er = SerializeEVModifications(e)
	if er != nil {
		return er
	}

	// create EV Combined
	er = SerializeEVCombined(e)
	if er != nil {
		return er
	}

	return nil
}

// SerializeEVPSM creates an ev serial with Evidence data
func SerializeEVPSM(e *Evidence) *err.Error {

	f, er := os.Create(sys.EvPSMBin())
	if er != nil {
		return &err.Error{Type: err.CannotCreateOutputFile, Class: err.FATA, Argument: er.Error()}
	}
	de := gob.NewEncoder(f)
	goberr := de.Encode(e.PSM)
	if goberr != nil {
		return &err.Error{Type: err.CannotSerializeData, Class: err.FATA, Argument: goberr.Error()}
	}
	f.Close()

	return nil
}

// SerializeEVIon creates an ev serial with Evidence data
func SerializeEVIon(e *Evidence) *err.Error {

	f, er := os.Create(sys.EvIonBin())
	if er != nil {
		return &err.Error{Type: err.CannotCreateOutputFile, Class: err.FATA, Argument: er.Error()}
	}
	de := gob.NewEncoder(f)
	goberr := de.Encode(e.Ions)
	if goberr != nil {
		return &err.Error{Type: err.CannotSerializeData, Class: err.FATA, Argument: goberr.Error()}
	}
	f.Close()

	return nil
}

// SerializeEVPeptides creates an ev serial with Evidence data
func SerializeEVPeptides(e *Evidence) *err.Error {

	f, er := os.Create(sys.EvPeptideBin())
	if er != nil {
		return &err.Error{Type: err.CannotCreateOutputFile, Class: err.FATA, Argument: er.Error()}
	}
	de := gob.NewEncoder(f)
	goberr := de.Encode(e.Peptides)
	if goberr != nil {
		return &err.Error{Type: err.CannotSerializeData, Class: err.FATA, Argument: goberr.Error()}
	}
	f.Close()

	return nil
}

// SerializeEVProteins creates an ev serial with Evidence data
func SerializeEVProteins(e *Evidence) *err.Error {

	f, er := os.Create(sys.EvProteinBin())
	if er != nil {
		return &err.Error{Type: err.CannotCreateOutputFile, Class: err.FATA, Argument: er.Error()}
	}
	de := gob.NewEncoder(f)
	goberr := de.Encode(e.Proteins)
	if goberr != nil {
		return &err.Error{Type: err.CannotSerializeData, Class: err.FATA, Argument: goberr.Error()}
	}
	f.Close()

	return nil
}

// SerializeEVMods creates an ev serial with Evidence data
func SerializeEVMods(e *Evidence) *err.Error {

	f, er := os.Create(sys.EvModificationsBin())
	if er != nil {
		return &err.Error{Type: err.CannotCreateOutputFile, Class: err.FATA, Argument: er.Error()}
	}
	de := gob.NewEncoder(f)
	goberr := de.Encode(e.Mods)
	if goberr != nil {
		return &err.Error{Type: err.CannotSerializeData, Class: err.FATA, Argument: goberr.Error()}
	}
	f.Close()

	return nil
}

// SerializeEVModifications creates an ev serial with Evidence data
func SerializeEVModifications(e *Evidence) *err.Error {

	f, er := os.Create(sys.EvModificationsEvBin())
	if er != nil {
		return &err.Error{Type: err.CannotCreateOutputFile, Class: err.FATA, Argument: er.Error()}
	}
	de := gob.NewEncoder(f)
	goberr := de.Encode(e.Modifications)
	if goberr != nil {
		return &err.Error{Type: err.CannotSerializeData, Class: err.FATA, Argument: goberr.Error()}
	}
	f.Close()

	return nil
}

// SerializeEVCombined creates an ev serial with Evidence data
func SerializeEVCombined(e *Evidence) *err.Error {

	f, er := os.Create(sys.EvCombinedBin())
	if er != nil {
		return &err.Error{Type: err.CannotCreateOutputFile, Class: err.FATA, Argument: er.Error()}
	}
	de := gob.NewEncoder(f)
	goberr := de.Encode(e.Combined)
	if goberr != nil {
		return &err.Error{Type: err.CannotSerializeData, Class: err.FATA, Argument: goberr.Error()}
	}
	f.Close()

	return nil
}

// Restore reads philosopher results files and restore the data sctructure
func (e *Evidence) Restore() error {

	file, _ := os.Open(sys.EvBin())

	dec := gob.NewDecoder(file)
	err := dec.Decode(&e)
	if err != nil {
		return errors.New("Could not restore Philosopher result. Please check file path")
	}

	return nil
}

// RestoreGranular reads philosopher results files and restore the data sctructure
func (e *Evidence) RestoreGranular() *err.Error {

	// PSM
	err := RestoreEVPSM(e)
	if err != nil {
		return err
	}

	// Ion
	err = RestoreEVIon(e)
	if err != nil {
		return err
	}

	// Peptide
	err = RestoreEVPeptide(e)
	if err != nil {
		return err
	}

	// Protein
	err = RestoreEVProtein(e)
	if err != nil {
		return err
	}

	// Mods
	err = RestoreEVMods(e)
	if err != nil {
		return err
	}

	// Modifications
	err = RestoreEVModifications(e)
	if err != nil {
		return err
	}

	// Combined
	err = RestoreEVCombined(e)
	if err != nil {
		return err
	}

	return nil
}

// RestoreEVPSM restores Ev PSM data
func RestoreEVPSM(e *Evidence) *err.Error {
	f, _ := os.Open(sys.EvPSMBin())
	d := gob.NewDecoder(f)
	er := d.Decode(&e.PSM)
	if er != nil {
		return &err.Error{Type: err.CannotRestoreGob, Class: err.FATA, Argument: er.Error()}
	}
	return nil
}

// RestoreEVIon restores Ev Ion data
func RestoreEVIon(e *Evidence) *err.Error {
	f, _ := os.Open(sys.EvIonBin())
	d := gob.NewDecoder(f)
	er := d.Decode(&e.Ions)
	if er != nil {
		return &err.Error{Type: err.CannotRestoreGob, Class: err.FATA, Argument: er.Error()}
	}
	return nil
}

// RestoreEVPeptide restores Ev Ion data
func RestoreEVPeptide(e *Evidence) *err.Error {
	f, _ := os.Open(sys.EvPeptideBin())
	d := gob.NewDecoder(f)
	er := d.Decode(&e.Peptides)
	if er != nil {
		return &err.Error{Type: err.CannotRestoreGob, Class: err.FATA, Argument: er.Error()}
	}
	return nil
}

// RestoreEVProtein restores Ev Protein data
func RestoreEVProtein(e *Evidence) *err.Error {
	f, _ := os.Open(sys.EvProteinBin())
	d := gob.NewDecoder(f)
	er := d.Decode(&e.Proteins)
	if er != nil {
		return &err.Error{Type: err.CannotRestoreGob, Class: err.FATA, Argument: er.Error()}
	}
	return nil
}

// RestoreEVMods restores Ev Mods data
func RestoreEVMods(e *Evidence) *err.Error {
	f, _ := os.Open(sys.EvModificationsBin())
	d := gob.NewDecoder(f)
	er := d.Decode(&e.Mods)
	if er != nil {
		return &err.Error{Type: err.CannotRestoreGob, Class: err.FATA, Argument: er.Error()}
	}
	return nil
}

// RestoreEVModifications restores Ev Mods data
func RestoreEVModifications(e *Evidence) *err.Error {
	f, _ := os.Open(sys.EvModificationsEvBin())
	d := gob.NewDecoder(f)
	er := d.Decode(&e.Modifications)
	if er != nil {
		return &err.Error{Type: err.CannotRestoreGob, Class: err.FATA, Argument: er.Error()}
	}
	return nil
}

// RestoreEVCombined restores Ev Mods data
func RestoreEVCombined(e *Evidence) *err.Error {
	f, _ := os.Open(sys.EvCombinedBin())
	d := gob.NewDecoder(f)
	er := d.Decode(&e.Combined)
	if er != nil {
		return &err.Error{Type: err.CannotRestoreGob, Class: err.FATA, Argument: er.Error()}
	}
	return nil
}

// RestoreGranularWithPath reads philosopher results files and restore the data sctructure
func (e *Evidence) RestoreGranularWithPath(p string) *err.Error {

	// PSM
	err := RestoreEVPSMWithPath(e, p)
	if err != nil {
		return err
	}

	// Ion
	err = RestoreEVIonWithPath(e, p)
	if err != nil {
		return err
	}

	// Peptide
	err = RestoreEVPeptideWithPath(e, p)
	if err != nil {
		return err
	}

	// Protein
	err = RestoreEVProteinWithPath(e, p)
	if err != nil {
		return err
	}

	// Mods
	err = RestoreEVModsWithPath(e, p)
	if err != nil {
		return err
	}

	// Modifications
	err = RestoreEVModificationsWithPath(e, p)
	if err != nil {
		return err
	}

	// Combined
	err = RestoreEVCombinedWithPath(e, p)
	if err != nil {
		return err
	}

	return nil
}

// RestoreEVPSMWithPath restores Ev PSM data
func RestoreEVPSMWithPath(e *Evidence, p string) *err.Error {

	path := sys.EvPSMBin()

	if strings.Contains(p, string(filepath.Separator)) {
		path = fmt.Sprintf("%s%s", p, sys.EvPSMBin())
	} else {
		path = fmt.Sprintf("%s%s%s", p, string(filepath.Separator), sys.EvPSMBin())
	}

	f, _ := os.Open(path)
	d := gob.NewDecoder(f)
	er := d.Decode(&e.PSM)
	if er != nil {
		return &err.Error{Type: err.CannotRestoreGob, Class: err.FATA, Argument: er.Error()}
	}
	return nil
}

// RestoreEVIonWithPath restores Ev Ion data
func RestoreEVIonWithPath(e *Evidence, p string) *err.Error {

	path := sys.EvIonBin()

	if strings.Contains(p, string(filepath.Separator)) {
		path = fmt.Sprintf("%s%s", p, sys.EvIonBin())
	} else {
		path = fmt.Sprintf("%s%s%s", p, string(filepath.Separator), sys.EvIonBin())
	}

	f, _ := os.Open(path)
	d := gob.NewDecoder(f)
	er := d.Decode(&e.Ions)
	if er != nil {
		return &err.Error{Type: err.CannotRestoreGob, Class: err.FATA, Argument: er.Error()}
	}
	return nil
}

// RestoreEVPeptideWithPath restores Ev Ion data
func RestoreEVPeptideWithPath(e *Evidence, p string) *err.Error {

	path := sys.EvPeptideBin()

	if strings.Contains(p, string(filepath.Separator)) {
		path = fmt.Sprintf("%s%s", p, sys.EvPeptideBin())
	} else {
		path = fmt.Sprintf("%s%s%s", p, string(filepath.Separator), sys.EvPeptideBin())
	}

	f, _ := os.Open(path)
	d := gob.NewDecoder(f)
	er := d.Decode(&e.Peptides)
	if er != nil {
		return &err.Error{Type: err.CannotRestoreGob, Class: err.FATA, Argument: er.Error()}
	}
	return nil
}

// RestoreEVProteinWithPath restores Ev Protein data
func RestoreEVProteinWithPath(e *Evidence, p string) *err.Error {

	path := sys.EvProteinBin()

	if strings.Contains(p, string(filepath.Separator)) {
		path = fmt.Sprintf("%s%s", p, sys.EvProteinBin())
	} else {
		path = fmt.Sprintf("%s%s%s", p, string(filepath.Separator), sys.EvProteinBin())
	}

	f, _ := os.Open(path)
	d := gob.NewDecoder(f)
	er := d.Decode(&e.Proteins)
	if er != nil {
		return &err.Error{Type: err.CannotRestoreGob, Class: err.FATA, Argument: er.Error()}
	}
	return nil
}

// RestoreEVModsWithPath restores Ev Mods data
func RestoreEVModsWithPath(e *Evidence, p string) *err.Error {

	path := sys.EvModificationsBin()

	if strings.Contains(p, string(filepath.Separator)) {
		path = fmt.Sprintf("%s%s", p, sys.EvModificationsBin())
	} else {
		path = fmt.Sprintf("%s%s%s", p, string(filepath.Separator), sys.EvModificationsBin())
	}

	f, _ := os.Open(path)
	d := gob.NewDecoder(f)
	er := d.Decode(&e.Mods)
	if er != nil {
		return &err.Error{Type: err.CannotRestoreGob, Class: err.FATA, Argument: er.Error()}
	}
	return nil
}

// RestoreEVModificationsWithPath restores Ev Mods data
func RestoreEVModificationsWithPath(e *Evidence, p string) *err.Error {

	path := sys.EvModificationsEvBin()

	if strings.Contains(p, string(filepath.Separator)) {
		path = fmt.Sprintf("%s%s", p, sys.EvModificationsEvBin())
	} else {
		path = fmt.Sprintf("%s%s%s", p, string(filepath.Separator), sys.EvModificationsEvBin())
	}

	f, _ := os.Open(path)
	d := gob.NewDecoder(f)
	er := d.Decode(&e.Modifications)
	if er != nil {
		return &err.Error{Type: err.CannotRestoreGob, Class: err.FATA, Argument: er.Error()}
	}
	return nil
}

// RestoreEVCombinedWithPath restores Ev Mods data
func RestoreEVCombinedWithPath(e *Evidence, p string) *err.Error {

	path := sys.EvCombinedBin()

	if strings.Contains(p, string(filepath.Separator)) {
		path = fmt.Sprintf("%s%s", p, sys.EvCombinedBin())
	} else {
		path = fmt.Sprintf("%s%s%s", p, string(filepath.Separator), sys.EvCombinedBin())
	}

	f, _ := os.Open(path)
	d := gob.NewDecoder(f)
	er := d.Decode(&e.Combined)
	if er != nil {
		return &err.Error{Type: err.CannotRestoreGob, Class: err.FATA, Argument: er.Error()}
	}
	return nil
}

// // RestoreWithPath reads philosopher results files and restore the data sctructure
// func (e *Evidence) RestoreWithPath(p string) error {
//
// 	var path string
//
// 	if strings.Contains(p, string(filepath.Separator)) {
// 		path = fmt.Sprintf("%s%s", p, sys.EvBin())
// 	} else {
// 		path = fmt.Sprintf("%s%s%s", p, string(filepath.Separator), sys.EvBin())
// 	}
//
// 	file, _ := os.Open(path)
//
// 	dec := gob.NewDecoder(file)
// 	err := dec.Decode(&e)
// 	if err != nil {
// 		return errors.New("Could not restore Philosopher result. Please check file path")
// 	}
//
// 	return nil
// }
