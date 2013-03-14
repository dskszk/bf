package main

import "os"
import "os/exec"
import "io"
import "strconv"

func main() {
	tmpd := os.TempDir() + "/bf"
	defer os.RemoveAll(tmpd)
	dstName := "a.out"
	switch len(os.Args) {
	case 1:
		os.Stdout.WriteString("bf src [dst]\n")
		return
	case 2:
		break
	default:
		dstName = os.Args[2]
	}
	srcName := os.Args[1]
	src, ferr := os.Open(srcName)
	if ferr != nil {
		os.Stdout.WriteString(ferr.Error() + "\n")
		return
	}
	os.Mkdir(tmpd, os.ModeDir|0755)
	dst, _ := os.Create(tmpd + "/a.S")
	if !compile(src, dst) {
		return
	}
	cmd := exec.Command("as", "-o", tmpd+"/a.o", tmpd+"/a.S")
	perr := cmd.Run()
	if perr != nil {
		os.Stdout.WriteString(perr.Error() + "\n")
		return
	}
	cmd = exec.Command("ld", "-o", dstName, tmpd+"/a.o")
	perr = cmd.Run()
	if perr != nil {
		os.Stdout.WriteString(perr.Error() + "\n")
	}
}

func compile(src, dst *os.File) bool {
	c, rerr := readByte(src)
	incp, incv := 0, 0
	st, sp := make([]int, 0), 0
	label := 0
	dst.WriteString(".text\n.globl _start\n_start:\nmovq $base,%rbx\n" +
		"movl $1,%edx\nxorl %r8d,%r8d\n")
	for rerr != io.EOF {
		switch c {
		case '>', '<':
			if incv != 0 {
				dst.WriteString(add(&incv, false))
			}
			switch c {
			case '>':
				incp++
			case '<':
				incp--
			}
		case '+', '-':
			if incp != 0 {
				dst.WriteString(add(&incp, true))
			}
			switch c {
			case '+':
				incv++
			case '-':
				incv--
			}
		case '.', ',', '[', ']':
			if incv != 0 {
				dst.WriteString(add(&incv, false))
			} else if incp != 0 {
				dst.WriteString(add(&incp, true))
			}
			switch c {
			case '.':
				dst.WriteString("xorl %eax,%eax\n" +
					"incl %eax\nmovl %eax,%edi\n" +
					"leaq (%rbx,%r8),%rsi\nsyscall\n")
			case ',':
				dst.WriteString("xorl %eax,%eax\n" +
					"movl %eax,%edi\n" +
					"leaq (%rbx,%r8),%rsi\nsyscall\n")
			case '[':
				dst.WriteString("cmpb $0,(%rbx,%r8)\n" +
					"jz .L" + strconv.Itoa(2*label+1) +
					"\n.L" + strconv.Itoa(2*label) +
					":\n")
				sp++
				if sp <= len(st) {
					st[sp-1] = label
				} else {
					st = append(st, label)
				}
				label++
			case ']':
				dst.WriteString("cmpb $0,(%rbx,%r8)\n" +
					"jnz .L" + strconv.Itoa(2*st[sp-1]) +
					"\n.L" + strconv.Itoa(2*st[sp-1]+1) +
					":\n")
				sp--
			}
		}
		c, rerr = readByte(src)
	}
	dst.WriteString("xorl %edi,%edi\nmovl %edi,%eax\nmovb $60,%al\n" +
		"syscall\n.bss\nbase: .skip 30000\n")
	src.Close()
	dst.Close()
	if sp != 0 {
		os.Stdout.WriteString("Brackets is not closed!\n")
		return false
	}
	return true
}

func readByte(f *os.File) (rune, error) {
	var b [1]byte
	_, err := f.Read(b[0:1])
	return rune(b[0]), err
}

func add(s *int, p bool) string {
	i := *s
	*s = 0
	switch {
	case i > 1:
		t := strconv.Itoa(i)
		if p {
			return "addl $" + t + ",%r8d\n"
		} else {
			return "addb $" + t + ",(%rbx,%r8)\n"
		}
	case i == 1:
		if p {
			return "incl %r8d\n"
		} else {
			return "incb (%rbx,%r8)\n"
		}
	case i == -1:
		if p {
			return "decl %r8d\n"
		} else {
			return "decb (%rbx,%r8)\n"
		}
	case i < -1:
		t := strconv.Itoa(-1 * i)
		if p {
			return "subl $" + t + ",%r8d\n"
		} else {
			return "subb $" + t + ",(%rbx,%r8)\n"
		}
	}
	return ""
}
