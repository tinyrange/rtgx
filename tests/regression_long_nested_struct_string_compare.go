package main

type rtg_example_com_regression_cmd_app_Box struct {
	Inner rtg_example_com_regression_cmd_app_Inner
}

type rtg_example_com_regression_cmd_app_Inner struct {
	Tag string
}

func rtg_example_com_regression_cmd_app_same(left rtg_example_com_regression_cmd_app_Box, right rtg_example_com_regression_cmd_app_Box) bool {
	return left.Inner.Tag == right.Inner.Tag
}

func appMain(args []string, env []string) int {
	left := rtg_example_com_regression_cmd_app_Box{Inner: rtg_example_com_regression_cmd_app_Inner{Tag: "b"}}
	right := rtg_example_com_regression_cmd_app_Box{Inner: rtg_example_com_regression_cmd_app_Inner{Tag: "b"}}
	if rtg_example_com_regression_cmd_app_same(left, right) {
		if left.Inner.Tag == right.Inner.Tag {
			print("PASS\n")
			return 0
		}
	}
	print("FAIL\n")
	return 0
}
