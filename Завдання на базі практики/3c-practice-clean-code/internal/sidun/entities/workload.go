package entities

// Workload — Value Object для навантаження викладача
type Workload struct {
	hours int
}

// WorkloadParams — об'єкт параметрів для обчислення навантаження
type WorkloadParams struct {
	Lectures    int
	Labs        int
	Practices   int
	IsHead      bool
	YearsActive int
}

const maxWorkloadHours = 40

// CalculateWorkload обчислює навантаження викладача за параметрами
func CalculateWorkload(params WorkloadParams) Workload {
	load := params.Lectures*2 + params.Labs + params.Practices

	if params.IsHead {
		load += 10
	}

	load += params.YearsActive / 5

	if load > maxWorkloadHours {
		load = maxWorkloadHours
	}

	return Workload{hours: load}
}

func (w Workload) Hours() int            { return w.hours }
func (w Workload) IsOverloaded() bool    { return w.hours >= maxWorkloadHours }
