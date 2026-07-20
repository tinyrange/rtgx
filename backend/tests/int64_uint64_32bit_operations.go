package main

func widePair(left uint64, right uint64) (uint64, uint64) {
	return left + right, left - right
}

func wideNamed(value uint64) (result uint64) {
	result = value + uint64(0x100000000)
	return
}

func wideArithmetic(left uint64, right uint64) uint64 {
	return left*right + left/right + left%right
}

func appMain(args []string) int {
	var low uint64 = 0xffffffff
	var two uint64 = 2
	var wide uint64 = 0x100000001
	var seven uint64 = 7
	var shift40 uint = 40
	var shift8 uint = 8

	if low+two != uint64(0x100000001) || wide-two != uint64(0xffffffff) {
		print("FAIL add/sub\n")
		return 1
	}
	if wide*seven != uint64(0x700000007) || uint64(0x700000007)/seven != wide || uint64(0x700000009)%seven != uint64(2) {
		print("FAIL mul/div/mod\n")
		return 1
	}
	if wideArithmetic(wide, seven) != uint64(0x724924930) {
		print("FAIL arithmetic call\n")
		return 1
	}

	var bits uint64 = 0x10000000055
	var mask uint64 = 0xff
	if bits&mask != uint64(0x55) || bits|uint64(0xaa) != uint64(0x100000000ff) || bits^uint64(0x05) != uint64(0x10000000050) || bits&^uint64(0x0f) != uint64(0x10000000050) {
		print("FAIL bitwise\n")
		return 1
	}
	if -wide != uint64(0xfffffffeffffffff) {
		print("FAIL unary\n")
		return 1
	}

	var one uint64 = 1
	if one<<shift40 != uint64(0x10000000000) || uint64(0x10000000000)>>shift8 != uint64(0x100000000) {
		print("FAIL shifts\n")
		return 1
	}
	var shift64 uint = 64
	var shift65 uint = 65
	if one<<shift64 != 0 || one<<shift65 != 0 || uint64(0xffffffffffffffff)>>shift64 != 0 {
		print("FAIL large shifts\n")
		return 1
	}

	var unsignedHigh uint64 = 0x8000000000000000
	var unsignedLow uint64 = 0x7fffffffffffffff
	if !(unsignedHigh > unsignedLow) {
		print("FAIL unsigned >\n")
		return 1
	}
	if unsignedHigh < unsignedLow {
		print("FAIL unsigned <\n")
		return 1
	}
	if unsignedHigh <= unsignedLow {
		print("FAIL unsigned <=\n")
		return 1
	}
	if !(unsignedHigh >= unsignedLow) {
		print("FAIL unsigned >=\n")
		return 1
	}
	var minusOne int64 = -1
	var positive int64 = 1
	var signedMin int64 = -0x7fffffffffffffff - 1
	if !(minusOne < positive) || minusOne >= positive || !(signedMin < minusOne) || signedMin > minusOne {
		print("FAIL signed compare\n")
		return 1
	}
	if minusOne>>shift40 != int64(-1) || signedMin>>shift40 != int64(-0x800000) {
		print("FAIL signed shift\n")
		return 1
	}
	var signedLeft int64 = -0x100000001
	var signedRight int64 = 7
	if signedLeft+signedRight != int64(-0xfffffffa) || signedLeft-signedRight != int64(-0x100000008) {
		print("FAIL signed add/sub\n")
		return 1
	}
	if signedLeft*signedRight != int64(-0x700000007) {
		print("FAIL signed mul\n")
		return 1
	}
	if signedLeft/signedRight != int64(-0x24924924) {
		print("FAIL signed div\n")
		return 1
	}
	if signedLeft%signedRight != int64(-5) {
		print("FAIL signed mod\n")
		return 1
	}

	var negative32 int32 = -7
	var max32 uint32 = 0xffffffff
	if int64(negative32) != int64(-7) || uint64(max32) != uint64(0xffffffff) || uint32(wide) != uint32(1) {
		print("FAIL conversions\n")
		return 1
	}
	sum, difference := widePair(wide, seven)
	if sum != uint64(0x100000008) || difference != uint64(0xfffffffa) || wideNamed(wide) != uint64(0x200000001) {
		print("FAIL returns\n")
		return 1
	}

	print("PASS\n")
	return 0
}
