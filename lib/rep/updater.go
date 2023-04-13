package rep

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/Nesvilab/philosopher/lib/dat"
	"github.com/Nesvilab/philosopher/lib/id"
	"github.com/Nesvilab/philosopher/lib/raz"
	"github.com/Nesvilab/philosopher/lib/uti"
)

// PeptideMap struct
type PeptideMap struct {
	Sequence  string
	IonForm   string
	Protein   string
	ProteinID string
	Gene      string
	Proteins  map[string]int
}

// UpdateNumberOfEnzymaticTermini collects the NTT from ProteinProphet
// and passes along to the final Protein structure.
func (evi *Evidence) UpdateNumberOfEnzymaticTermini(decoyTag string) {

	// restore the original prot.xml output
	var p id.ProtIDList
	p.Restore()

	// collect the updated ntt for each peptide-protein pair
	//var nttPeptidetoProptein = make(map[string]uint8)
	type k struct {
		a string
		b string
	}
	var nttPeptidetoProptein = make(map[k]uint8)

	for _, i := range p {
		for _, j := range i.PeptideIons {
			if !strings.Contains(i.ProteinName, decoyTag) {
				nttPeptidetoProptein[k{j.PeptideSequence, i.ProteinName}] = j.NumberOfEnzymaticTermini
			}
		}
	}

	for i := range evi.PSM {
		if ntt, ok := nttPeptidetoProptein[k{evi.PSM[i].Peptide, evi.PSM[i].Protein}]; ok {
			evi.PSM[i].NumberOfEnzymaticTermini = ntt
		}
	}
}

// UpdateIonStatus pushes back to ion and psm evideces the uniqueness and razorness status of each peptide and ion
func (evi *Evidence) UpdateIonStatus(decoyTag string) {

	var uniqueIons = make(map[id.IonFormType]bool)
	var razorIons = make(map[id.IonFormType]string)
	var uniquePeptides = make(map[string]string)
	var razorPeptides = make(map[string]string)

	for i := 0; i < len(evi.Proteins); i++ {
		for _, j := range evi.Proteins[i].TotalPeptideIons {

			if j.IsUnique {
				uniqueIons[j.IonForm()] = true
				uniquePeptides[j.Sequence] = evi.Proteins[i].PartHeader
			}

			if j.IsURazor {
				razorIons[j.IonForm()] = evi.Proteins[i].PartHeader
				razorPeptides[j.Sequence] = evi.Proteins[i].PartHeader
			}
		}
	}

	for i := range evi.PSM {
		// the decoy tag checking is a failsafe mechanism to avoid proteins
		// with real complex razor case decisions to pass downstream
		// wrong classifications. If by any chance the protein gets assigned to
		// a razor decoy, this mechanism avoids the replacement

		rp, rOK := razorIons[evi.PSM[i].IonForm()]
		if rOK {

			evi.PSM[i].IsURazor = true

			// we found cases where the peptide maps to both target and decoy but is
			// assigned as razor to the decoy. the IF statement below replaces the
			// decoy by the target but it was removed because in some cases the protein
			// does not pass the FDR filtering.

			evi.PSM[i].MappedProteins[evi.PSM[i].Protein] = evi.PSM[i].PrevAA + "#" + evi.PSM[i].NextAA

			// recover prevAA-nextAA
			altPrt := evi.PSM[i].MappedProteins[rp]
			prevNext := strings.Split(altPrt, "#")

			delete(evi.PSM[i].MappedProteins, rp)

			evi.PSM[i].Protein = rp
			evi.PSM[i].PrevAA = prevNext[0]
			evi.PSM[i].PrevAA = prevNext[1]
		}

		if !evi.PSM[i].IsURazor {
			sp, sOK := razorPeptides[evi.PSM[i].Peptide]
			if sOK {

				evi.PSM[i].IsURazor = true

				// we found cases where the peptide maps to both target and decoy but is
				// assigned as razor to the decoy. the IF statement below replaces the
				// decoy by the target but it was removed because in some cases the protein
				// does not pass the FDR filtering.

				evi.PSM[i].MappedProteins[evi.PSM[i].Protein] = evi.PSM[i].PrevAA + "#" + evi.PSM[i].NextAA

				// recover prevAA-nextAA
				altPrt := evi.PSM[i].MappedProteins[sp]
				prevNext := strings.Split(altPrt, "#")

				delete(evi.PSM[i].MappedProteins, sp)

				evi.PSM[i].Protein = sp
				evi.PSM[i].PrevAA = prevNext[0]
				evi.PSM[i].PrevAA = prevNext[1]

				if strings.Contains(sp, decoyTag) {
					evi.PSM[i].IsDecoy = true
				}
			}

			_, uOK := uniqueIons[evi.PSM[i].IonForm()]
			if uOK {
				evi.PSM[i].IsUnique = true
			}

			uniquePeptides[evi.PSM[i].Peptide] = evi.PSM[i].Protein
		}
	}

	for i := range evi.Ions {
		rp, rOK := razorIons[evi.Ions[i].IonForm()]
		if rOK {

			evi.Ions[i].IsURazor = true

			evi.Ions[i].MappedProteins[evi.Ions[i].Protein] = 0
			delete(evi.Ions[i].MappedProteins, rp)
			evi.Ions[i].Protein = rp

			if strings.Contains(rp, decoyTag) {
				evi.Ions[i].IsDecoy = true
			}

		}
		_, uOK := uniqueIons[evi.Ions[i].IonForm()]
		if uOK {
			evi.Ions[i].IsUnique = true
		} else {
			evi.Ions[i].IsUnique = false
		}
	}

	for i := range evi.Peptides {
		rp, rOK := razorPeptides[evi.Peptides[i].Sequence]
		if rOK {

			evi.Peptides[i].IsURazor = true

			evi.Peptides[i].MappedProteins[evi.Peptides[i].Protein] = 0
			delete(evi.Peptides[i].MappedProteins, rp)
			evi.Peptides[i].Protein = rp

			if strings.Contains(rp, decoyTag) {
				evi.Peptides[i].IsDecoy = true
			}

		}
		_, uOK := uniquePeptides[evi.Peptides[i].Sequence]
		if uOK {
			evi.Peptides[i].IsUnique = true
		} else {
			evi.Peptides[i].IsUnique = false
		}
	}
}

// UpdateIonModCount counts how many times each ion is observed modified and not modified
func (evi *Evidence) UpdateIonModCount() {

	// recreate the ion list from the main report object
	var AllIons = make(map[id.IonFormType]struct{})
	var ModIons = make(map[id.IonFormType]int)
	var UnModIons = make(map[id.IonFormType]int)

	for i := 0; i < len(evi.Ions); i++ {
		AllIons[evi.Ions[i].IonForm()] = struct{}{}
		ModIons[evi.Ions[i].IonForm()] = 0
		UnModIons[evi.Ions[i].IonForm()] = 0
	}

	// range over PSMs looking for modified and not modified evidences
	// if they exist on the ions map, get the numbers
	for i := 0; i < len(evi.PSM); i++ {

		// check the map
		_, ok := AllIons[evi.PSM[i].IonForm()]
		if ok {

			if evi.PSM[i].Massdiff >= -0.99 && evi.PSM[i].Massdiff <= 0.99 {
				UnModIons[evi.PSM[i].IonForm()]++
			} else {
				ModIons[evi.PSM[i].IonForm()]++
			}

		}
	}
}

// SyncPSMToProteins forces the synchronization between the filtered proteins, and the remaining structures.
func (evi *Evidence) SyncPSMToProteins(decoy string) {

	var totalSpc = make(map[string][]id.SpectrumType, len(evi.PSM))
	var uniqueSpc = make(map[string][]id.SpectrumType, len(evi.PSM))
	var razorSpc = make(map[string][]id.SpectrumType, len(evi.PSM))

	var totalPeptides = make(map[string][]string, len(evi.PSM))
	var uniquePeptides = make(map[string][]string, len(evi.PSM))
	var razorPeptides = make(map[string][]string, len(evi.PSM))

	var proteinIndex = make(map[string]struct{})

	for i := 0; i < len(evi.Proteins); i++ {
		proteinIndex[evi.Proteins[i].PartHeader] = struct{}{}
	}

	for i := 0; i < len(evi.PSM); i++ {

		// Total
		totalSpc[evi.PSM[i].Protein] = append(totalSpc[evi.PSM[i].Protein], evi.PSM[i].SpectrumFileName())
		totalPeptides[evi.PSM[i].Protein] = append(totalPeptides[evi.PSM[i].Protein], evi.PSM[i].Peptide)

		for j := range evi.PSM[i].MappedProteins {
			totalSpc[j] = append(totalSpc[j], evi.PSM[i].SpectrumFileName())
			totalPeptides[j] = append(totalPeptides[j], evi.PSM[i].Peptide)
		}

		if evi.PSM[i].IsUnique {
			uniqueSpc[evi.PSM[i].Protein] = append(uniqueSpc[evi.PSM[i].Protein], evi.PSM[i].SpectrumFileName())
			uniquePeptides[evi.PSM[i].Protein] = append(uniquePeptides[evi.PSM[i].Protein], evi.PSM[i].Peptide)
		}

		if evi.PSM[i].IsURazor {
			razorSpc[evi.PSM[i].Protein] = append(razorSpc[evi.PSM[i].Protein], evi.PSM[i].SpectrumFileName())
			razorPeptides[evi.PSM[i].Protein] = append(razorPeptides[evi.PSM[i].Protein], evi.PSM[i].Peptide)
		}
	}

	for k, v := range totalPeptides {
		totalPeptides[k] = uti.RemoveDuplicateStrings(v)
	}

	for k, v := range uniquePeptides {
		uniquePeptides[k] = uti.RemoveDuplicateStrings(v)
	}

	for k, v := range razorPeptides {
		razorPeptides[k] = uti.RemoveDuplicateStrings(v)
	}

	for i := range evi.Proteins {

		evi.Proteins[i].SupportingSpectra = make(map[id.SpectrumType]int)
		evi.Proteins[i].TotalSpC = 0
		evi.Proteins[i].UniqueSpC = 0
		evi.Proteins[i].URazorSpC = 0
		evi.Proteins[i].TotalPeptides = make(map[string]int)
		evi.Proteins[i].UniquePeptides = make(map[string]int)
		evi.Proteins[i].URazorPeptides = make(map[string]int)

		if v, ok := totalSpc[evi.Proteins[i].PartHeader]; ok {
			evi.Proteins[i].TotalSpC += len(v)
			for _, j := range v {
				evi.Proteins[i].SupportingSpectra[j]++
			}
		}

		if v, ok := totalPeptides[evi.Proteins[i].PartHeader]; ok {
			for _, j := range v {
				evi.Proteins[i].TotalPeptides[j]++
			}
		}

		if v, ok := uniqueSpc[evi.Proteins[i].PartHeader]; ok {
			evi.Proteins[i].UniqueSpC += len(v)
		}

		if v, ok := uniquePeptides[evi.Proteins[i].PartHeader]; ok {
			for _, j := range v {
				evi.Proteins[i].UniquePeptides[j]++
			}
		}

		if v, ok := razorSpc[evi.Proteins[i].PartHeader]; ok {
			evi.Proteins[i].URazorSpC += len(v)
		}

		if v, ok := razorPeptides[evi.Proteins[i].PartHeader]; ok {
			for _, j := range v {
				evi.Proteins[i].URazorPeptides[j]++
			}
		}
	}

	{
		proteinIndex = make(map[string]struct{}, len(evi.Proteins))
		newProteins := make([]int, 0, len(evi.Proteins))
		for idx, i := range evi.Proteins {
			if len(i.SupportingSpectra) > 0 {
				proteinIndex[i.PartHeader] = struct{}{}
				newProteins = append(newProteins, idx)
			}
		}
		for idx, i := range newProteins {
			evi.Proteins[idx] = evi.Proteins[i]
		}
		evi.Proteins = evi.Proteins[:len(newProteins)]
	}
	{
		newPSM := make([]int, 0, len(evi.PSM))
		for idx, i := range evi.PSM {
			if _, ok := proteinIndex[i.Protein]; ok {
				newPSM = append(newPSM, idx)
			}
		}
		for idx, i := range newPSM {
			evi.PSM[idx] = evi.PSM[i]
		}
		evi.PSM = evi.PSM[:len(newPSM)]
	}
	{
		newIons := make([]int, 0, len(evi.Ions))
		for idx, i := range evi.Ions {
			if _, ok := proteinIndex[i.Protein]; ok {
				newIons = append(newIons, idx)
			}
		}
		for idx, i := range newIons {
			evi.Ions[idx] = evi.Ions[i]
		}
		evi.Ions = evi.Ions[:len(newIons)]
	}
	{
		newPeptides := make([]int, 0, len(evi.Peptides))
		for idx, i := range evi.Peptides {
			if _, ok := proteinIndex[i.Protein]; ok {
				newPeptides = append(newPeptides, idx)
			}
		}
		for idx, i := range newPeptides {
			evi.Peptides[idx] = evi.Peptides[i]
		}
		evi.Peptides = evi.Peptides[:len(newPeptides)]
	}
}

// SyncPSMToPeptides forces the synchronization between the filtered peptides, and the remaining structures.
func (evi Evidence) SyncPSMToPeptides(decoy string) Evidence {

	var spectra = make(map[string][]id.SpectrumType)

	for i := 0; i < len(evi.PSM); i++ {
		if !evi.PSM[i].IsDecoy {
			spectra[evi.PSM[i].Peptide] = append(spectra[evi.PSM[i].Peptide], evi.PSM[i].SpectrumFileName())
		}
	}

	for i := range evi.Peptides {

		evi.Peptides[i].Spc = 0
		evi.Peptides[i].Spectra = make(map[id.SpectrumType]uint8)

		if v, ok := spectra[evi.Peptides[i].Sequence]; ok {

			//evi.Peptides[i].IsDecoy = false

			for _, j := range v {
				evi.Peptides[i].Spectra[j]++
			}
			evi.Peptides[i].Spc = len(v)

		}
	}

	return evi
}

// SyncPSMToPeptideIons forces the synchronization between the filtered ions, and the remaining structures.
func (evi Evidence) SyncPSMToPeptideIons(decoy string) Evidence {

	var spectra = make(map[id.IonFormType][]id.SpectrumType)

	for i := 0; i < len(evi.PSM); i++ {
		if !evi.PSM[i].IsDecoy {
			spectra[evi.PSM[i].IonForm()] = append(spectra[evi.PSM[i].IonForm()], evi.PSM[i].SpectrumFileName())
		}
	}

	for i := range evi.Ions {

		evi.Ions[i].Spectra = make(map[id.SpectrumType]int)

		v, ok := spectra[evi.Ions[i].IonForm()]
		if ok {

			//evi.Ions[i].IsDecoy = false

			for _, j := range v {
				evi.Ions[i].Spectra[j]++
			}
		}
	}

	return evi
}

// UpdateLayerswithDatabase will fix the protein and gene assignments based on the database data
func (evi *Evidence) UpdateLayerswithDatabase(dbBin, decoyTag string) {

	type liteRecord struct {
		ID          string
		EntryName   string
		GeneNames   string
		Description string
		Sequence    string
	}
	var recordMap = make(map[string]liteRecord)

	{
		var dtb dat.Base

		if len(dbBin) == 0 {
			dtb.Restore()
		} else {
			dtb.RestoreWithPath(dbBin)
		}

		for _, j := range dtb.Records {
			recordMap[j.PartHeader] = liteRecord{j.ID, j.EntryName, j.GeneNames, strings.TrimSpace(j.ProteinName), j.Sequence}
		}
	}

	var proteinStart = make(map[string]int)
	var proteinEnd = make(map[string]int)

	replacerIL := strings.NewReplacer("L", "I")
	for i := range evi.PSM {

		rec := recordMap[evi.PSM[i].Protein]
		evi.PSM[i].ProteinID = rec.ID
		evi.PSM[i].EntryName = rec.EntryName
		evi.PSM[i].GeneName = rec.GeneNames
		evi.PSM[i].ProteinDescription = rec.Description

		// ensure the assignment is a decoy
		if strings.HasPrefix(evi.PSM[i].Protein, decoyTag) {
			evi.PSM[i].IsDecoy = true
		}

		// update mapped genes
		for k := range evi.PSM[i].MappedProteins {
			if !strings.Contains(k, decoyTag) {
				evi.PSM[i].MappedGenes[recordMap[k].GeneNames] = struct{}{}
			}
		}

		var adjustStart = 0
		var adjustEnd = 0

		peptide := replacerIL.Replace(evi.PSM[i].Peptide)

		if evi.PSM[i].PrevAA != "-" && len(evi.PSM[i].PrevAA) == 1 {
			peptide = replacerIL.Replace(evi.PSM[i].PrevAA) + peptide
			adjustStart = +2
		}

		if evi.PSM[i].PrevAA == "-" && len(evi.PSM[i].PrevAA) == 1 {
			adjustStart = +1
		}

		if evi.PSM[i].NextAA != "-" && len(evi.PSM[i].NextAA) == 1 {
			peptide = peptide + replacerIL.Replace(evi.PSM[i].NextAA)
			adjustEnd = -1
		}

		// map the peptide to the protein
		mstart := strings.Index(replacerIL.Replace(rec.Sequence), peptide)
		mend := mstart + len(peptide)

		evi.PSM[i].ProteinStart = mstart + adjustStart
		evi.PSM[i].ProteinEnd = mend + adjustEnd

		proteinStart[evi.PSM[i].Peptide] = evi.PSM[i].ProteinStart
		proteinEnd[evi.PSM[i].Peptide] = evi.PSM[i].ProteinEnd

		// {
		// 	proteinStart[evi.PSM[i].Peptide] = mstart
		// 	proteinEnd[evi.PSM[i].Peptide] = mend

		// 	seq := recordMap[evi.PSM[i].Protein].Sequence

		// 	fmt.Println(evi.PSM[i].Peptide)
		// 	fmt.Println(peptide)

		// 	fmt.Println(mstart)

		// 	mapPep := seq[mstart:mend]
		// 	fmt.Println(mapPep)

		// 	fmt.Println("")
		// }

		// map the flanking aminoacids
		flanks := regexp.MustCompile(`(\w{0,7})` + regexp.QuoteMeta(peptide) + `(\w{0,7})`)
		f := flanks.FindAllStringSubmatch(replacerIL.Replace(rec.Sequence), -1)

		var left string
		var right string

		if f != nil {

			match := f[0]

			if len(match) >= 1 && len(match[1]) > 0 {
				left = fmt.Sprintf("%s%s.", match[1], evi.PSM[i].PrevAA)
			} else {
				left = "."
			}

			if len(match) >= 2 && len(match[2]) > 0 {
				right = fmt.Sprintf(".%s%s", evi.PSM[i].NextAA, match[2])
			} else {
				right = "."
			}

			evi.PSM[i].ExtendedPeptide = left + evi.PSM[i].Peptide + right

		} else {
			evi.PSM[i].ExtendedPeptide = "." + evi.PSM[i].Peptide + "."
		}
	}

	for i, ion := range evi.Ions {
		id := ion.Protein
		if ion.IsDecoy {
			id = strings.Replace(id, decoyTag, "", 1)
		}

		rec, ok := recordMap[id]
		if !ok {
			// handle error: record not found in map
			continue
		}

		tmp := &evi.Ions[i]

		tmp.ProteinID = rec.ID
		tmp.EntryName = rec.EntryName
		tmp.GeneName = rec.GeneNames
		tmp.ProteinDescription = rec.Description

		// ensure the assignment is a decoy
		if strings.HasPrefix(evi.Ions[i].Protein, decoyTag) {
			evi.Ions[i].IsDecoy = true
		}

		tmp.ProteinStart, tmp.ProteinEnd = proteinStart[tmp.Sequence], proteinEnd[tmp.Sequence]

		// update mapped genes
		for k := range ion.MappedProteins {
			if strings.Contains(k, decoyTag) {
				continue
			}

			geneName := recordMap[k].GeneNames
			tmp.MappedGenes[geneName] = struct{}{}
		}
	}

	for i := range evi.Peptides {

		id := evi.Peptides[i].Protein
		if evi.Peptides[i].IsDecoy {
			id = strings.Replace(id, decoyTag, "", 1)
		}

		rec := recordMap[id]
		evi.Peptides[i].ProteinID = rec.ID
		evi.Peptides[i].EntryName = rec.EntryName
		evi.Peptides[i].GeneName = rec.GeneNames
		evi.Peptides[i].ProteinDescription = rec.Description

		// ensure the assignment is a decoy
		if strings.HasPrefix(evi.Peptides[i].Protein, decoyTag) {
			evi.Peptides[i].IsDecoy = true
		}

		if seq, ok := proteinStart[evi.Peptides[i].Sequence]; ok {
			evi.Peptides[i].ProteinStart = seq
		}
		if seq, ok := proteinEnd[evi.Peptides[i].Sequence]; ok {
			evi.Peptides[i].ProteinEnd = seq
		}

		// update mapped genes
		for k := range evi.Peptides[i].MappedProteins {
			if !strings.Contains(k, decoyTag) {
				evi.Peptides[i].MappedGenes[recordMap[k].GeneNames] = struct{}{}
			}
		}
	}

}

// UpdateSupportingSpectra pushes back from PSM to Protein the new supporting spectra from razor results
func (evi *Evidence) UpdateSupportingSpectra() {

	var ptSupSpec = make(map[string][]id.SpectrumType)
	var uniqueSpec = make(map[id.IonFormType][]id.SpectrumType)
	var razorSpec = make(map[id.IonFormType][]id.SpectrumType)

	var totalPeptides = make(map[string][]string)
	var uniquePeptides = make(map[string][]string)
	var razorPeptides = make(map[string][]string)

	for i := 0; i < len(evi.PSM); i++ {

		_, ok := ptSupSpec[evi.PSM[i].Protein]
		if !ok {
			ptSupSpec[evi.PSM[i].Protein] = append(ptSupSpec[evi.PSM[i].Protein], evi.PSM[i].SpectrumFileName())
		}

		if evi.PSM[i].IsUnique {
			_, ok := uniqueSpec[evi.PSM[i].IonForm()]
			if !ok {
				uniqueSpec[evi.PSM[i].IonForm()] = append(uniqueSpec[evi.PSM[i].IonForm()], evi.PSM[i].SpectrumFileName())
			}
		}

		if evi.PSM[i].IsURazor {
			_, ok := razorSpec[evi.PSM[i].IonForm()]
			if !ok {
				razorSpec[evi.PSM[i].IonForm()] = append(razorSpec[evi.PSM[i].IonForm()], evi.PSM[i].SpectrumFileName())
			}
		}
	}

	for i := 0; i < len(evi.Peptides); i++ {

		totalPeptides[evi.Peptides[i].Protein] = append(totalPeptides[evi.Peptides[i].Protein], evi.Peptides[i].Sequence)
		for j := range evi.Peptides[i].MappedProteins {
			totalPeptides[j] = append(totalPeptides[j], evi.Peptides[i].Sequence)
		}

		if evi.Peptides[i].IsUnique {
			uniquePeptides[evi.Peptides[i].Protein] = append(uniquePeptides[evi.Peptides[i].Protein], evi.Peptides[i].Sequence)
		}

		if evi.Peptides[i].IsURazor {
			razorPeptides[evi.Peptides[i].Protein] = append(razorPeptides[evi.Peptides[i].Protein], evi.Peptides[i].Sequence)
		}
	}

	for k, v := range totalPeptides {
		totalPeptides[k] = uti.RemoveDuplicateStrings(v)
	}

	for k, v := range uniquePeptides {
		uniquePeptides[k] = uti.RemoveDuplicateStrings(v)
	}

	for k, v := range razorPeptides {
		razorPeptides[k] = uti.RemoveDuplicateStrings(v)
	}

	for i := range evi.Proteins {
		for j := range evi.Proteins[i].TotalPeptideIons {

			if len(evi.Proteins[i].TotalPeptideIons[j].Spectra) == 0 {
				delete(evi.Proteins[i].TotalPeptideIons, j)
			}
		}
	}

	for i := range evi.Proteins {

		v, ok := ptSupSpec[evi.Proteins[i].PartHeader]
		if ok {
			for _, j := range v {
				evi.Proteins[i].SupportingSpectra[j] = 0
			}
		}

		for k := range evi.Proteins[i].TotalPeptideIons {

			Up, UOK := uniqueSpec[evi.Proteins[i].TotalPeptideIons[k].IonForm()]
			if UOK && evi.Proteins[i].TotalPeptideIons[k].IsUnique {
				for _, l := range Up {
					evi.Proteins[i].TotalPeptideIons[k].Spectra[l] = 0
				}
			}

			Rp, ROK := razorSpec[evi.Proteins[i].TotalPeptideIons[k].IonForm()]
			if ROK && evi.Proteins[i].TotalPeptideIons[k].IsURazor {
				for _, l := range Rp {
					evi.Proteins[i].TotalPeptideIons[k].Spectra[l] = 0
				}
			}

		}

		vTP, okTP := totalPeptides[evi.Proteins[i].PartHeader]
		if okTP {
			for _, j := range vTP {
				evi.Proteins[i].TotalPeptides[j]++
			}
		}

		vuP, okuP := uniquePeptides[evi.Proteins[i].PartHeader]
		if okuP {
			for _, j := range vuP {
				evi.Proteins[i].UniquePeptides[j]++
			}
		}

		vRP, okRP := razorPeptides[evi.Proteins[i].PartHeader]
		if okRP {
			for _, j := range vRP {
				evi.Proteins[i].URazorPeptides[j]++
			}
		}

	}
}

// UpdatePeptideModCount counts how many times each peptide is observed modified and not modified
func (evi *Evidence) UpdatePeptideModCount() {

	// recreate the ion list from the main report object
	var all = make(map[string]int)
	var mod = make(map[string]int)
	var unmod = make(map[string]int)

	for i := 0; i < len(evi.Peptides); i++ {
		all[evi.Peptides[i].Sequence] = 0
		mod[evi.Peptides[i].Sequence] = 0
		unmod[evi.Peptides[i].Sequence] = 0
	}

	// range over PSMs looking for modified and not modified evidences
	// if they exist on the ions map, get the numbers
	for i := 0; i < len(evi.PSM); i++ {

		_, ok := all[evi.PSM[i].Peptide]
		if ok {

			if evi.PSM[i].Massdiff >= -0.99 && evi.PSM[i].Massdiff <= 0.99 {
				unmod[evi.PSM[i].Peptide]++
			} else {
				mod[evi.PSM[i].Peptide]++
			}

		}
	}

	for i := range evi.Peptides {

		v1, ok1 := unmod[evi.Peptides[i].Sequence]
		if ok1 {
			evi.Peptides[i].UnModifiedObservations = v1
		}

		v2, ok2 := mod[evi.Peptides[i].Sequence]
		if ok2 {
			evi.Peptides[i].ModifiedObservations = v2
		}

	}
}

// CalculateProteinCoverage calcualtes the peptide coverage for each protein
func (evi *Evidence) CalculateProteinCoverage() {

	replacerIL := strings.NewReplacer("L", "I")

	for p := range evi.Proteins {

		var pep []string
		evi.Proteins[p].Coverage = -1

		// the original sequence used as template
		seqA := replacerIL.Replace(evi.Proteins[p].Sequence)

		// the replaced sequence
		seqB := ""

		for i := range evi.Proteins[p].TotalPeptides {
			pep = append(pep, replacerIL.Replace(i))
		}

		if len(pep) > 0 {

			seqB = seqA

			for _, i := range pep {

				re := regexp.MustCompile(i)
				match := re.FindAllStringIndex(seqA, -1)

				for j := 0; j <= len(match)-1; j++ {
					seqB = seqB[:match[j][0]] + string(strings.Repeat("X", len(i))) + seqB[match[j][1]:]
				}
			}

			count := strings.Count(seqB, "X")

			cent := (float64(count) / float64(evi.Proteins[p].Length)) * 100

			evi.Proteins[p].Coverage = float32(cent)

		} else {
			evi.Proteins[p].Coverage = float32(0)
		}
	}
}

// ApplyRazorAssignment propagates the razor assignment to the data
func (evi *Evidence) ApplyRazorAssignment(decoyTag string) {

	var razor raz.RazorMap = make(map[string]raz.RazorCandidate)
	razor.Restore(false)

	for i := range evi.PSM {

		v, ok := razor[evi.PSM[i].Peptide]
		if ok {

			if evi.PSM[i].IsUnique {
				evi.PSM[i].IsURazor = true
			}

			if len(v.MappedProtein) == 0 {

				evi.PSM[i].IsURazor = false

			} else {

				evi.PSM[i].IsURazor = true

				evi.PSM[i].MappedProteins[evi.PSM[i].Protein] = evi.PSM[i].PrevAA + "#" + evi.PSM[i].NextAA

				// TODO recover prev-next
				altPrt := evi.PSM[i].MappedProteins[v.MappedProtein]
				prevNext := strings.Split(altPrt, "#")

				delete(evi.PSM[i].MappedProteins, v.MappedProtein)

				evi.PSM[i].Protein = v.MappedProtein

				// to avoid decoys
				if len(prevNext) > 1 {
					evi.PSM[i].PrevAA = prevNext[0]
					evi.PSM[i].NextAA = prevNext[1]
				}

			}
		}

		if strings.HasPrefix(evi.PSM[i].Protein, decoyTag) {
			evi.PSM[i].IsDecoy = true
		}
	}

	for i := range evi.Ions {

		v, ok := razor[evi.Ions[i].Sequence]

		if ok {

			if evi.Ions[i].IsUnique {
				evi.Ions[i].IsURazor = true
			}

			if len(v.MappedProtein) == 0 {

				evi.Ions[i].IsURazor = false

			} else {

				evi.Ions[i].IsURazor = true

				evi.Ions[i].MappedProteins[evi.Ions[i].Protein]++
				delete(evi.Ions[i].MappedProteins, v.MappedProtein)
				evi.Ions[i].Protein = v.MappedProtein

			}
		}

		if strings.HasPrefix(evi.Ions[i].Protein, decoyTag) {
			evi.Ions[i].IsDecoy = true
		}
	}

	for i := range evi.Peptides {

		v, ok := razor[evi.Peptides[i].Sequence]

		if ok {

			if evi.Peptides[i].IsUnique {
				evi.Peptides[i].IsURazor = true
			}

			if len(v.MappedProtein) == 0 {

				evi.Peptides[i].IsURazor = false

			} else {

				evi.Peptides[i].IsURazor = true

				evi.Peptides[i].MappedProteins[evi.Peptides[i].Protein]++
				delete(evi.Peptides[i].MappedProteins, v.MappedProtein)
				evi.Peptides[i].Protein = v.MappedProtein

			}
		}

		if strings.HasPrefix(evi.Peptides[i].Protein, decoyTag) {
			evi.Peptides[i].IsDecoy = true
		}
	}
}
