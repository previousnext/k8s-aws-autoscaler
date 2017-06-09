package main

func scaler(cpu, memory int, instance *InstanceType) int {
	// How many instances does the memory require.
	// Given memory is the most popular, we assume it as the desired.
	d := memory / instance.Memory
	c := cpu / instance.CPU

	if c > d {
		d = c
	}

	// We increase the "desired" by one because our division chops off a
	// certain percentage.
	//   eg. 2.45 = 2 (but we still need compute for the .45)
	d++

	return d
}
