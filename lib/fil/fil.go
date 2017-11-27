package fil

import (
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/prvst/philosopher/lib/clas"
	"github.com/prvst/philosopher/lib/data"
	"github.com/prvst/philosopher/lib/meta"
	"github.com/prvst/philosopher/lib/quan"
	"github.com/prvst/philosopher/lib/rep"
	"github.com/prvst/philosopher/lib/sys"
	"github.com/prvst/philosopher/lib/utils"
	"github.com/prvst/philosopher/lib/xml"
)

// Filter object
type Filter struct {
	meta.Data
	Phi      string
	Pex      string
	Pox      string
	Tag      string
	Con      string
	Psmfdr   float64
	Pepfdr   float64
	Ionfdr   float64
	Ptfdr    float64
	ProtProb float64
	PepProb  float64
	Ptconf   string
	RepProt  string
	Save     string
	Database string
	TopPep   bool
	Model    bool
	RepPSM   bool
	Razor    bool
	Picked   bool
	Seq      bool
	Mapmods  bool
}

// New constructor
func New() Filter {

	var o Filter
	var m meta.Data
	m.Restore(sys.Meta())

	o.UUID = m.UUID
	o.Distro = m.Distro
	o.Home = m.Home
	o.MetaFile = m.MetaFile
	o.MetaDir = m.MetaDir
	o.DB = m.DB
	o.Temp = m.Temp
	o.TimeStamp = m.TimeStamp
	o.OS = m.OS
	o.Arch = m.Arch

	return o
}

// Run executes the Filter processing
func (f *Filter) Run(psmFDR, pepFDR, ionFDR, ptFDR, pepProb, protProb float64, isPicked, isRazor, mapmod bool) error {

	e := rep.New()
	var pepxml xml.PepXML
	var pep xml.PepIDList
	var pro xml.ProtIDList
	var err error

	logrus.Info("Processing peptide identification files")

	pepid, err := readPepXMLInput(f.Pex, f.Tag, f.Model)
	if err != nil {
		return err
	}

	err = processPeptideIdentifications(pepid, f.Tag, psmFDR, pepFDR, ionFDR)
	if err != nil {
		return err
	}

	if len(f.Pox) > 0 {

		protXML, proerr := readProtXMLInput(sys.MetaDir(), f.Pox, f.Tag)
		if proerr != nil {
			return proerr
		}

		err = processProteinIdentifications(protXML, ptFDR, pepProb, protProb, isPicked, isRazor)
		if err != nil {
			return err
		}
		//protXML = xml.ProtXML{}

		if f.Seq == true {

			// sequential analysis
			// filtered psm list and filtered prot list
			pep.Restore("psm")
			pro.Restore()
			err = sequentialFDRControl(pep, pro, psmFDR, pepFDR, ionFDR, f.Tag)
			if err != nil {
				return err
			}
			pep = nil
			pro = nil

		} else {

			// two-dimensional analysis
			// complete pep list and filtered mirror-image prot list
			pepxml.Restore()
			pro.Restore()
			err = twoDFDRFilter(pepxml.PeptideIdentification, pro, psmFDR, pepFDR, ionFDR, f.Tag)
			if err != nil {
				return err
			}
			pepxml = xml.PepXML{}
			pro = nil

		}

	}

	var dtb data.Base
	dtb.Restore()
	if len(dtb.Records) < 1 {
		return errors.New("Database data not available, interrupting processing")
	}

	logrus.Info("Post processing identifications")

	// restoring for the modifications
	var pxml xml.PepXML
	pxml.Restore()
	e.Mods.DefinedModAminoAcid = pxml.DefinedModAminoAcid
	e.Mods.DefinedModMassDiff = pxml.DefinedModMassDiff
	pxml = xml.PepXML{}

	var psm xml.PepIDList
	psm.Restore("psm")
	e.AssemblePSMReport(psm, f.Tag)
	psm = nil

	// evaluate modifications in data set
	if mapmod == true {
		logrus.Info("Mapping modifications")
		e.MapMassDiffToUniMod()

		logrus.Info("Processing modifications")
		e.AssembleModificationReport()
	}

	var ion xml.PepIDList
	ion.Restore("ion")
	e.AssembleIonReport(ion, f.Tag)
	ion = nil

	var pept xml.PepIDList
	pept.Restore("pep")
	e.AssemblePeptideReport(pept, f.Tag)
	pept = nil

	// evaluate modifications in data set
	if mapmod == true {
		e.UpdateIonModCount()
		e.UpdatePeptideModCount()
	}

	logrus.Info("Processing Protein Inference")
	pro.Restore()
	err = e.AssembleProteinReport(pro, f.Tag)
	if err != nil {
		return err
	}
	pro = nil

	// ADD ERROR CASES
	logrus.Info("Correcting PSM to Protein mappings")
	e.UpdateIndistinguishableProteinLists()

	// ADD ERROR CASES
	logrus.Info("Mapping Ion status to PSMs")
	e.UpdateIonStatus()

	logrus.Info("Calculating Spectral Counts")
	e, cerr := quan.CalculateSpectralCounts(e)
	if cerr != nil {
		return cerr
	}

	e.Meta.DecoyTag = f.Tag

	cerr = e.SerializeGranular()
	if cerr != nil {
		return cerr
	}

	return nil
}

// readPepXMLInput reads one or more fies and organize the data into PSM list
func readPepXMLInput(xmlFile, decoyTag string, models bool) (xml.PepIDList, error) {

	var files []string
	var pepIdent xml.PepIDList
	var definedModMassDiff = make(map[float64]float64)
	var definedModAminoAcid = make(map[float64]string)

	if strings.Contains(xmlFile, "pep.xml") || strings.Contains(xmlFile, "pepXML") {
		files = append(files, xmlFile)
	} else {
		glob := fmt.Sprintf("%s/*pep.xml", xmlFile)
		list, _ := filepath.Glob(glob)

		if len(list) == 0 {
			return pepIdent, errors.New("No pepXML files found, check your files and try again")
		}

		for _, i := range list {
			absPath, _ := filepath.Abs(i)
			files = append(files, absPath)
		}

	}

	for _, i := range files {
		var p xml.PepXML
		p.DecoyTag = decoyTag
		e := p.Read(i)
		if e != nil {
			return nil, e
		}

		// print models
		if models == true {
			if strings.EqualFold(p.Prophet, "interprophet") {
				logrus.Error("Cannot print models for interprophet files")
			} else {
				logrus.Info("Printing models")
				temp, _ := sys.GetTemp()
				go p.ReportModels(temp, filepath.Base(i))
				time.Sleep(time.Second * 3)
			}
		}

		pepIdent = append(pepIdent, p.PeptideIdentification...)

		for k, v := range p.DefinedModAminoAcid {
			definedModAminoAcid[k] = v
		}

		for k, v := range p.DefinedModMassDiff {
			definedModMassDiff[k] = v
		}

	}

	// create a "fake" global pepXML comprising all data
	var pepXML xml.PepXML
	pepXML.DecoyTag = decoyTag
	pepXML.PeptideIdentification = pepIdent
	pepXML.DefinedModAminoAcid = definedModAminoAcid
	pepXML.DefinedModMassDiff = definedModMassDiff

	// promoting Spectra that matches to both decoys and targets to TRUE hits
	pepXML.PromoteProteinIDs()

	// serialize all pep files
	sort.Sort(pepXML.PeptideIdentification)
	pepXML.Serialize()

	return pepIdent, nil
}

// processPeptideIdentifications reads and process pepXML
func processPeptideIdentifications(p xml.PepIDList, decoyTag string, psm, peptide, ion float64) error {

	var err error

	// report charge profile
	var t, d int

	t, d, _ = chargeProfile(p, 1, decoyTag)
	logrus.WithFields(logrus.Fields{
		"target": t,
		"decoy":  d,
	}).Info("1+ Charge profile")

	t, d, _ = chargeProfile(p, 2, decoyTag)
	logrus.WithFields(logrus.Fields{
		"target": t,
		"decoy":  d,
	}).Info("2+ Charge profile")

	t, d, _ = chargeProfile(p, 3, decoyTag)
	logrus.WithFields(logrus.Fields{
		"target": t,
		"decoy":  d,
	}).Info("3+ Charge profile")

	t, d, _ = chargeProfile(p, 4, decoyTag)
	logrus.WithFields(logrus.Fields{
		"target": t,
		"decoy":  d,
	}).Info("4+ Charge profile")

	t, d, _ = chargeProfile(p, 5, decoyTag)
	logrus.WithFields(logrus.Fields{
		"target": t,
		"decoy":  d,
	}).Info("5+ Charge profile")

	t, d, _ = chargeProfile(p, 6, decoyTag)
	logrus.WithFields(logrus.Fields{
		"target": t,
		"decoy":  d,
	}).Info("6+ Charge profile")

	uniqPsms := getUniquePSMs(p)
	uniqPeps := getUniquePeptides(p)
	uniqIons := getUniquePeptideIons(p)

	logrus.WithFields(logrus.Fields{
		"psms":     len(p),
		"peptides": len(uniqPeps),
		"ions":     len(uniqIons),
	}).Info("Database search results")

	filteredPSM, err := pepXMLFDRFilter(uniqPsms, psm, "PSM", decoyTag)
	if err != nil {
		logrus.Fatal(err)
	}
	filteredPSM.Serialize("psm")

	filteredPeptides, err := pepXMLFDRFilter(uniqPeps, peptide, "Peptide", decoyTag)
	if err != nil {
		logrus.Fatal(err)
	}
	filteredPeptides.Serialize("pep")

	filteredIons, err := pepXMLFDRFilter(uniqIons, ion, "Ion", decoyTag)
	if err != nil {
		logrus.Fatal(err)
	}
	filteredIons.Serialize("ion")

	return nil
}

// chargeProfile ...
func chargeProfile(p xml.PepIDList, charge uint8, decoyTag string) (t, d int, err error) {

	for _, i := range p {
		if i.AssumedCharge == charge {
			if strings.Contains(i.Protein, decoyTag) {
				d++
			} else {
				t++
			}
		}
	}

	if t < 1 || d < 1 {
		err = errors.New("Invalid charge state count")
	}

	return t, d, err
}

//getUniquePSMs selects only unique pepetide ions for the given data stucture
func getUniquePSMs(p xml.PepIDList) map[string]xml.PepIDList {

	uniqMap := make(map[string]xml.PepIDList)

	for _, i := range p {
		uniqMap[i.Spectrum] = append(uniqMap[i.Spectrum], i)
	}

	return uniqMap
}

//getUniquePeptideIons selects only unique pepetide ions for the given data stucture
func getUniquePeptideIons(p xml.PepIDList) map[string]xml.PepIDList {

	uniqMap := ExtractIonsFromPSMs(p)

	return uniqMap
}

// ExtractIonsFromPSMs takes a pepidlist and transforms into an ion map
func ExtractIonsFromPSMs(p xml.PepIDList) map[string]xml.PepIDList {

	uniqMap := make(map[string]xml.PepIDList)

	for _, i := range p {
		ion := fmt.Sprintf("%s#%d#%.4f", i.Peptide, i.AssumedCharge, i.CalcNeutralPepMass)
		uniqMap[ion] = append(uniqMap[ion], i)
	}

	// organize id list by score
	for _, v := range uniqMap {
		sort.Sort(v)
	}

	return uniqMap
}

// getUniquePeptides selects only unique pepetide for the given data stucture
func getUniquePeptides(p xml.PepIDList) map[string]xml.PepIDList {

	uniqMap := make(map[string]xml.PepIDList)

	for _, i := range p {
		uniqMap[string(i.Peptide)] = append(uniqMap[string(i.Peptide)], i)
	}

	// organize id list by score
	for _, v := range uniqMap {
		sort.Sort(v)
	}

	return uniqMap
}

// pepXMLFDRFilter applies FDR filtering at the PSM level
func pepXMLFDRFilter(input map[string]xml.PepIDList, targetFDR float64, level, decoyTag string) (xml.PepIDList, error) {

	//var msg string
	var targets float64
	var decoys float64
	var calcFDR float64
	var list xml.PepIDList
	var peplist xml.PepIDList
	var minProb float64 = 10
	var err error

	if strings.EqualFold(level, "PSM") {

		// move all entries to list and count the number of targets and decoys
		for _, i := range input {
			for _, j := range i {
				if clas.IsDecoyPSM(j, decoyTag) {
					decoys++
				} else {
					targets++
				}
				list = append(list, j)
			}
		}

	} else if strings.EqualFold(level, "Peptide") {

		// 0 index means the one with highest score
		for _, i := range input {
			peplist = append(peplist, i[0])
		}

		for i := range peplist {
			if clas.IsDecoyPSM(peplist[i], decoyTag) {
				decoys++
			} else {
				targets++
			}
			list = append(list, peplist[i])
		}

	} else if strings.EqualFold(level, "Ion") {

		// 0 index means the one with highest score
		for _, i := range input {
			peplist = append(peplist, i[0])
		}

		for i := range peplist {
			if clas.IsDecoyPSM(peplist[i], decoyTag) {
				decoys++
			} else {
				targets++
			}
			list = append(list, peplist[i])
		}

	} else {
		err = errors.New("Error applying FDR score; unknown level")
	}

	sort.Sort(list)

	var scoreMap = make(map[float64]float64)
	limit := (len(list) - 1)
	for j := limit; j >= 0; j-- {
		_, ok := scoreMap[list[j].Probability]
		if !ok {
			scoreMap[list[j].Probability] = (decoys / targets)
		}
		if clas.IsDecoyPSM(list[j], decoyTag) {
			decoys--
		} else {
			targets--
		}
	}

	var keys []float64
	for k := range scoreMap {
		keys = append(keys, k)
	}

	sort.Sort(sort.Reverse(sort.Float64Slice(keys)))

	var probList = make(map[float64]uint8)
	for i := range keys {

		//f := fmt.Sprintf("%.2f", scoreMap[keys[i]]*100)
		//f := utils.Round(scoreMap[keys[i]]*100, 5, 2)
		//fmt.Println(keys[i], "\t", scoreMap[keys[i]], "\t", utils.ToFixed(scoreMap[keys[i]], 4), "\t", f, "\t", targetFDR)

		if utils.ToFixed(scoreMap[keys[i]], 4) <= targetFDR {
			probList[keys[i]] = 0
			minProb = keys[i]
			calcFDR = utils.ToFixed(scoreMap[keys[i]], 4)
		}

	}

	var cleanlist xml.PepIDList
	decoys = 0
	targets = 0

	for i := range list {
		_, ok := probList[list[i].Probability]
		if ok {
			cleanlist = append(cleanlist, list[i])
			if clas.IsDecoyPSM(list[i], decoyTag) {
				decoys++
			} else {
				targets++
			}
		}
	}

	msg := fmt.Sprintf("Converged to %.2f %% FDR with %0.f %ss", (calcFDR * 100), targets, level)
	logrus.WithFields(logrus.Fields{
		"decoy":     decoys,
		"total":     (targets + decoys),
		"threshold": minProb,
	}).Info(msg)

	return cleanlist, err
}

// readProtXMLInput reads one or more fies and organize the data into PSM list
func readProtXMLInput(meta, xmlFile, decoyTag string) (xml.ProtXML, error) {

	var protXML xml.ProtXML

	err := protXML.Read(xmlFile)
	if err != nil {
		return protXML, err
	}

	protXML.DecoyTag = decoyTag

	protXML.PromoteProteinIDs()

	protXML.Serialize()

	return protXML, nil
}

// processProteinIdentifications checks if pickedFDR ar razor options should be applied to given data set, if they do,
// the inputed protXML data is processed before filtered.
func processProteinIdentifications(p xml.ProtXML, ptFDR, pepProb, protProb float64, isPicked, isRazor bool) error {

	var err error
	var pid xml.ProtIDList

	// tagget / decoy / threshold
	t, d, _ := proteinProfile(p)
	logrus.WithFields(logrus.Fields{
		"target": t,
		"decoy":  d,
	}).Info("Protein inference results")

	// applies pickedFDR algorithm
	if isPicked == true {
		p = pickedFDR(p)
	}

	// applies razor algorithm
	if isRazor == true {
		p, err = RazorFilter(p)
		if err != nil {
			return err
		}
	}

	// run the FDR filter for proteins
	pid, err = ProtXMLFilter(p, ptFDR, pepProb, protProb, isPicked, isRazor)
	if err != nil {
		return err
	}

	// save results on meta folder
	pid.Serialize()

	return nil
}

// proteinProfile ...
func proteinProfile(p xml.ProtXML) (t, d int, err error) {

	for _, i := range p.Groups {
		for _, j := range i.Proteins {
			if clas.IsDecoyProtein(j, p.DecoyTag) {
				d++
			} else {
				t++
			}
		}
	}

	return t, d, err
}

// Picked employs the picked FDR strategy
func pickedFDR(p xml.ProtXML) xml.ProtXML {

	// var appMap = make(map[string]int)
	var targetMap = make(map[string]float64)
	var decoyMap = make(map[string]float64)
	var recordMap = make(map[string]int)

	// collect all proteins from every group
	for _, i := range p.Groups {
		for _, j := range i.Proteins {
			if clas.IsDecoyProtein(j, p.DecoyTag) {
				decoyMap[string(j.ProteinName)] = j.PeptideIons[0].InitialProbability
			} else {
				targetMap[string(j.ProteinName)] = j.PeptideIons[0].InitialProbability
			}
		}
	}

	// check unique targets
	for k := range targetMap {
		iKey := fmt.Sprintf("%s%s", p.DecoyTag, k)
		_, ok := decoyMap[iKey]
		if !ok {
			recordMap[k] = 1
		}
	}

	// check unique decoys
	for k := range decoyMap {
		iKey := strings.Replace(k, p.DecoyTag, "", -1)
		_, ok := targetMap[iKey]
		if !ok {
			recordMap[k] = 1
		}
	}

	// check paired observations
	for k, v := range targetMap {
		iKey := fmt.Sprintf("%s%s", p.DecoyTag, k)
		vok, ok := decoyMap[iKey]
		if ok {
			if vok > v {
				recordMap[k] = 0
				recordMap[iKey] = 1
			} else if v > vok {
				recordMap[k] = 1
				recordMap[iKey] = 0
			} else {
				recordMap[k] = 1
				recordMap[iKey] = 1
			}
		}
	}

	// collect all proteins from every group
	for i := range p.Groups {
		for j := range p.Groups[i].Proteins {
			v, ok := recordMap[string(p.Groups[i].Proteins[j].ProteinName)]
			if ok {
				p.Groups[i].Proteins[j].Picked = v
			}
		}
	}

	return p
}

// RazorFilter classifies peptides as razor
func RazorFilter(p xml.ProtXML) (xml.ProtXML, error) {

	//var razorMap = make(map[string]string)
	var razorMap = make(map[string][]string)
	var refMap = make(map[string][]string)
	var pepMap = make(map[string]string)
	var checkRazor = make(map[string]uint8)

	// create reference entries by collapsing into a single string all necessary information about the peptide-to-protein assignment
	for i := range p.Groups {
		for j := range p.Groups[i].Proteins {
			for k := range p.Groups[i].Proteins[j].PeptideIons {

				ref := fmt.Sprintf("%d#%s#%s#%s#%f#%f#%f#%d#%.4f",
					p.Groups[i].GroupNumber,
					string(p.Groups[i].Proteins[j].GroupSiblingID),
					string(p.Groups[i].Proteins[j].ProteinName),
					string(p.Groups[i].Proteins[j].PeptideIons[k].PeptideSequence),
					p.Groups[i].Proteins[j].PeptideIons[k].InitialProbability,
					p.Groups[i].Proteins[j].PeptideIons[k].Weight,
					p.Groups[i].Proteins[j].PeptideIons[k].GroupWeight,
					p.Groups[i].Proteins[j].PeptideIons[k].Charge,
					p.Groups[i].Proteins[j].PeptideIons[k].CalcNeutralPepMass,
				)

				refMap[string(p.Groups[i].Proteins[j].PeptideIons[k].PeptideSequence)] = append(refMap[string(p.Groups[i].Proteins[j].PeptideIons[k].PeptideSequence)], ref)
				pepMap[string(p.Groups[i].Proteins[j].PeptideIons[k].PeptideSequence)] = ""

			}
		}
	}

	// for each unique peptide sequence
	for k := range pepMap {

		var gw float64
		var w float64
		var mgw []string

		// retrieve the list of references based on the peptide sequence
		v, ok := refMap[k]
		if ok {

			// for each reference in the list
			for i := range v {

				pep := strings.Split(v[i], "#")

				weight, err := strconv.ParseFloat(pep[5], 64)
				if err != nil {
					return p, err
				}

				groupWeight, err := strconv.ParseFloat(pep[6], 64)
				if err != nil {
					return p, err
				}

				cnpm, err := strconv.ParseFloat(pep[8], 64)
				if err != nil {
					return p, err
				}

				pepCheck := fmt.Sprintf("%s#%s#%.4f", pep[3], pep[7], cnpm)

				// references with weight > 0.5 are easy cases, and clearly assinged as razor
				if weight > 0.5 {

					mgw = append(mgw, v[i])
					_, ok := checkRazor[pepCheck]
					if !ok {
						mgw = append(mgw, v[i])
						checkRazor[pepCheck] = 1
					}

				} else {

					if groupWeight > gw {
						gw = groupWeight
						w = weight
						_, ok := checkRazor[pepCheck]
						if !ok {
							mgw = append(mgw, v[i])
							checkRazor[pepCheck] = 1
						}

					} else if groupWeight == gw {
						if weight > w {
							w = weight

							_, ok := checkRazor[pepCheck]
							if !ok {
								mgw = append(mgw, v[i])
								checkRazor[pepCheck] = 1
							}

						} else if weight == w {
							w = weight

							_, ok := checkRazor[pepCheck]
							if !ok {
								mgw = append(mgw, v[i])
								checkRazor[pepCheck] = 1
							}

						}
					}

				}

			}
		}
		razorMap[k] = mgw
	}

	for i := range p.Groups {
		for j := range p.Groups[i].Proteins {
			for k := range p.Groups[i].Proteins[j].PeptideIons {

				ref := fmt.Sprintf("%d#%s#%s#%s#%f#%f#%f#%d#%.4f",
					p.Groups[i].GroupNumber,
					string(p.Groups[i].Proteins[j].GroupSiblingID),
					string(p.Groups[i].Proteins[j].ProteinName),
					string(p.Groups[i].Proteins[j].PeptideIons[k].PeptideSequence),
					p.Groups[i].Proteins[j].PeptideIons[k].InitialProbability,
					p.Groups[i].Proteins[j].PeptideIons[k].Weight,
					p.Groups[i].Proteins[j].PeptideIons[k].GroupWeight,
					p.Groups[i].Proteins[j].PeptideIons[k].Charge,
					p.Groups[i].Proteins[j].PeptideIons[k].CalcNeutralPepMass)

				v, ok := razorMap[string(p.Groups[i].Proteins[j].PeptideIons[k].PeptideSequence)]
				if ok {

					for _, l := range v {
						if strings.EqualFold(ref, l) {
							p.Groups[i].Proteins[j].PeptideIons[k].Razor = 1
							p.Groups[i].Proteins[j].HasRazor = true
							break
						}
					}
				}

			}
		}
	}

	// mark as razor all peptides in the reference map
	for i := range p.Groups {
		for j := range p.Groups[i].Proteins {
			var r float64
			for k := range p.Groups[i].Proteins[j].PeptideIons {
				if p.Groups[i].Proteins[j].PeptideIons[k].Razor == 1 || p.Groups[i].Proteins[j].PeptideIons[k].IsUnique {
					if p.Groups[i].Proteins[j].PeptideIons[k].InitialProbability > r {
						r = p.Groups[i].Proteins[j].PeptideIons[k].InitialProbability
					}
				}
			}
			p.Groups[i].Proteins[j].TopPepProb = r
		}
	}

	return p, nil
}

// ProtXMLFilter filters the protein list under a specific fdr
func ProtXMLFilter(p xml.ProtXML, targetFDR, pepProb, protProb float64, isPicked, isRazor bool) (xml.ProtIDList, error) {

	//var proteinIDs ProtIDList
	var list xml.ProtIDList
	var targets float64
	var decoys float64
	var calcFDR float64
	var minProb float64 = 10
	var err error

	// collect all proteins from every group
	for i := range p.Groups {
		for j := range p.Groups[i].Proteins {

			if isRazor == true {

				if isPicked == true {
					if p.Groups[i].Proteins[j].Picked == 1 && p.Groups[i].Proteins[j].HasRazor == true {
						list = append(list, p.Groups[i].Proteins[j])
					}
				} else {
					if p.Groups[i].Proteins[j].HasRazor == true {
						list = append(list, p.Groups[i].Proteins[j])
					}
				}

			} else {

				if isPicked == true {
					if p.Groups[i].Proteins[j].Probability >= protProb && p.Groups[i].Proteins[j].Picked == 1 {
						list = append(list, p.Groups[i].Proteins[j])
					}

				} else {
					if p.Groups[i].Proteins[j].TopPepProb >= pepProb && p.Groups[i].Proteins[j].Probability >= protProb {
						list = append(list, p.Groups[i].Proteins[j])
					}
				}

			}

		}
	}

	for i := range list {
		if clas.IsDecoyProtein(list[i], p.DecoyTag) {
			decoys++
		} else {
			targets++
		}
	}

	sort.Sort(&list)

	// from botttom to top, classify every protein block with a given fdr score
	// the score is only calculates to the first (last) protein in each block
	// proteins with the same score, get the same fdr value.
	var scoreMap = make(map[float64]float64)
	for j := (len(list) - 1); j >= 0; j-- {
		_, ok := scoreMap[list[j].TopPepProb]
		if !ok {
			scoreMap[list[j].TopPepProb] = (decoys / targets)
		}

		if clas.IsDecoyProtein(list[j], p.DecoyTag) {
			decoys--
		} else {
			targets--
		}
	}

	var keys []float64
	for k := range scoreMap {
		keys = append(keys, k)
	}

	sort.Sort(sort.Reverse(sort.Float64Slice(keys)))

	var curProb = 10.0
	var curScore = 0.0
	var probArray []float64
	var probList = make(map[float64]uint8)

	for i := range keys {

		// for inspections
		//f := utils.Round(scoreMap[keys[i]]*100, 5, 2)
		//fmt.Println(keys[i], "\t", scoreMap[keys[i]], "\t", utils.ToFixed(scoreMap[keys[i]], 4), "\t", f)
		//fmt.Println(keys[i], "\t", scoreMap[keys[i]], "\t", utils.ToFixed(scoreMap[keys[i]], 4), "\t", f, "\t", targetFDR)

		probArray = append(probArray, keys[i])

		if utils.ToFixed(scoreMap[keys[i]], 4) <= targetFDR {
			probList[keys[i]] = 0
			minProb = keys[i]
			calcFDR = scoreMap[keys[i]]
			if keys[i] < curProb {
				curProb = keys[i]
			}
			if scoreMap[keys[i]] > curScore {
				curScore = scoreMap[keys[i]]
			}
		}

	}

	if curProb == 10 {
		msgProb := fmt.Sprintf("The protein FDR filter didn't reached the desired threshold of %.4f, try a higher threshold using the --prot parameter", targetFDR)
		err = errors.New(msgProb)
	}

	fmtScore := utils.ToFixed(curScore, 4)

	// for inspections
	//fmt.Println("curscore:", curScore, "\t", "fmtScore:", fmtScore, "\t", "targetfdr:", targetFDR)

	if curScore < targetFDR && fmtScore != targetFDR && probArray[len(probArray)-1] != curProb {

		for i := 0; i <= len(probArray); i++ {

			if probArray[i] == curProb {
				probList[probArray[i+1]] = 0
				minProb = probArray[i+1]
				calcFDR = scoreMap[probArray[i+1]]
				if probArray[i+1] < curProb {
					curProb = probArray[i+1]
				}
				if scoreMap[probArray[i+1]] > curScore {
					curScore = scoreMap[probArray[i+1]]
				}
				break
			}

		}

	}

	// for inspections
	//fmt.Println("curscore:", curScore, "\t", "fmtScore:", fmtScore, "\t", "targetfdr:", targetFDR)

	var cleanlist xml.ProtIDList
	for i := range list {
		_, ok := probList[list[i].TopPepProb]
		if ok {
			cleanlist = append(cleanlist, list[i])
			if clas.IsDecoyProtein(list[i], p.DecoyTag) {
				decoys++
			} else {
				targets++
			}
		}
	}

	msg := fmt.Sprintf("Converged to %.2f %% FDR with %0.f Proteins", (calcFDR * 100), targets)
	logrus.WithFields(logrus.Fields{
		"decoy":     decoys,
		"total":     (targets + decoys),
		"threshold": minProb,
	}).Info(msg)

	return cleanlist, err
}

// sequentialFDRControl estimates FDR levels by applying a second filter where all
// proteins from the protein filtered list are matched against filtered PSMs
func sequentialFDRControl(pep xml.PepIDList, pro xml.ProtIDList, psm, peptide, ion float64, decoyTag string) error {

	extPep := extractPSMfromPepXML(pep, pro)

	// organize enties by score (probability or expectation)
	sort.Sort(extPep)

	uniqPsms := getUniquePSMs(extPep)
	uniqPeps := getUniquePeptides(extPep)
	uniqIons := getUniquePeptideIons(extPep)

	logrus.WithFields(logrus.Fields{
		"psms":     len(uniqPsms),
		"peptides": len(uniqPeps),
		"ions":     len(uniqIons),
	}).Info("Applying sequential FDR estimation")

	filteredPSM, err := pepXMLFDRFilter(uniqPsms, psm, "PSM", decoyTag)
	if err != nil {
		logrus.Fatal(err)
	}
	filteredPSM.Serialize("psm")

	filteredPeptides, err := pepXMLFDRFilter(uniqPeps, peptide, "Peptide", decoyTag)
	if err != nil {
		logrus.Fatal(err)
	}
	filteredPeptides.Serialize("pep")

	filteredIons, err := pepXMLFDRFilter(uniqIons, ion, "Ion", decoyTag)
	if err != nil {
		logrus.Fatal(err)
	}
	filteredIons.Serialize("ion")

	return nil
}

// twoDFDRFilter estimates FDR levels by applying a second filter by regenerating
// a protein list with decoys from protXML and pepXML.
func twoDFDRFilter(pep xml.PepIDList, pro xml.ProtIDList, psm, peptide, ion float64, decoyTag string) error {

	// filter protein list at given FDR level and regenerate protein list by adding pairing decoys
	//logrus.Info("Creating mirror image from filtered protein list")
	mirrorProteinList := mirrorProteinList(pro, decoyTag)

	// get new protein list profile
	//logrus.Info(protxml.ProteinProfileWithList(mirrorProteinList, pa.Tag, pa.Con))
	t, d, _ := proteinProfileWithList(mirrorProteinList, decoyTag)
	logrus.WithFields(logrus.Fields{
		"target": t,
		"decoy":  d,
	}).Info("2D FDR estimation: Protein mirror image")

	// get PSM from the original pepXML using protein REGENERATED protein list, using protein names
	extPep := extractPSMfromPepXML(pep, mirrorProteinList)

	// organize enties by score (probability or expectation)
	sort.Sort(extPep)

	uniqPsms := getUniquePSMs(extPep)
	uniqPeps := getUniquePeptides(extPep)
	uniqIons := getUniquePeptideIons(extPep)

	logrus.WithFields(logrus.Fields{
		"psms":     len(uniqPsms),
		"peptides": len(uniqPeps),
		"ions":     len(uniqIons),
	}).Info("Second filtering results")

	filteredPSM, err := pepXMLFDRFilter(uniqPsms, psm, "PSM", decoyTag)
	if err != nil {
		logrus.Fatal(err)
	}
	filteredPSM.Serialize("psm")

	filteredPeptides, err := pepXMLFDRFilter(uniqPeps, peptide, "Peptide", decoyTag)
	if err != nil {
		logrus.Fatal(err)
	}
	filteredPeptides.Serialize("pep")

	filteredIons, err := pepXMLFDRFilter(uniqIons, ion, "Ion", decoyTag)
	if err != nil {
		logrus.Fatal(err)
	}
	filteredIons.Serialize("ion")

	return nil
}

// extractPSMfromPepXML retrieves all psm from protxml that maps into pepxml files
// using protein names from <protein> and <alternative_proteins> tags
func extractPSMfromPepXML(peplist xml.PepIDList, pro xml.ProtIDList) xml.PepIDList {

	var protmap = make(map[string]uint16)
	var filterMap = make(map[string]xml.PeptideIdentification)
	var output xml.PepIDList

	// get all protein names from protxml
	for _, i := range pro {
		protmap[string(i.ProteinName)] = 0
	}

	for _, i := range peplist {

		_, ptTag := protmap[string(i.Protein)]
		if ptTag {
			filterMap[string(i.Spectrum)] = i
			protmap[string(i.Protein)]++
		} else {
			for _, j := range i.AlternativeProteins {
				_, altTag := protmap[j]
				if altTag {
					filterMap[string(i.Spectrum)] = i
					protmap[string(j)]++
				}
			}
		}

	}

	// // match protein names to <protein> tag on pepxml
	// for j := range peplist {
	// 	_, ok := protmap[string(peplist[j].Protein)]
	// 	if ok {
	// 		filterMap[string(peplist[j].Spectrum)] = peplist[j]
	// 	}
	// }
	//
	// // match protein names to <alternative_proteins> tag on pepxml
	// for m := range peplist {
	// 	for n := range peplist[m].AlternativeProteins {
	// 		_, ok := protmap[peplist[m].AlternativeProteins[n]]
	// 		if ok {
	// 			filterMap[string(peplist[m].Spectrum)] = peplist[m]
	// 		}
	// 	}
	// }

	for _, v := range filterMap {
		output = append(output, v)
	}

	return output
}

// mirrorProteinList takes a filtered list and regenerate the correspondedn decoys
func mirrorProteinList(p xml.ProtIDList, decoyTag string) xml.ProtIDList {

	var targets = make(map[string]uint8)
	var decoys = make(map[string]uint8)

	// get filtered list
	var list xml.ProtIDList
	for _, i := range p {
		if !clas.IsDecoyProtein(i, decoyTag) {
			list = append(list, i)
		}
	}

	// get the list of identified taget proteins
	for _, i := range p {
		if clas.IsDecoy(i.ProteinName, decoyTag) {
			decoys[i.ProteinName] = 0
		} else {
			targets[i.ProteinName] = 0
		}
	}

	// collect all original protein ids in case we need to put them on mirror list
	var refMap = make(map[string]xml.ProteinIdentification)
	for _, i := range p {
		refMap[i.ProteinName] = i
	}

	// add decoys correspondent to the given targets.
	// first check if the oposite list doesn't have an entry already.
	// if not, search for the mirror entry on the original list, if found
	// move it to the mirror list, otherwise add fake entry.
	for _, k := range list {
		decoy := decoyTag + k.ProteinName
		v, ok := refMap[decoy]
		if ok {
			list = append(list, v)
		} else {
			var pt xml.ProteinIdentification
			pt.ProteinName = decoy
			list = append(list, pt)
		}
	}

	return list
}

// proteinProfileWithList ...
func proteinProfileWithList(list []xml.ProteinIdentification, decoyTag string) (t, d int, err error) {

	for i := range list {
		if clas.IsDecoyProtein(list[i], decoyTag) {
			d++
		} else {
			t++
		}
	}

	return t, d, err
}
