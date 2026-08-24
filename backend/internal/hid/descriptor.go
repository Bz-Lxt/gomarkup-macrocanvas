package hid

// CompositeReportDescriptor is a legal HID Report Descriptor for a
// boot-compatible keyboard + mouse composite device (USB HID).
func CompositeReportDescriptor() []byte {
	return []byte{
		// Keyboard collection
		0x05, 0x01, // Usage Page (Generic Desktop)
		0x09, 0x06, // Usage (Keyboard)
		0xA1, 0x01, // Collection (Application)
		0x85, 0x01, //   Report ID (1)
		0x05, 0x07, //   Usage Page (Keyboard)
		0x19, 0xE0, //   Usage Minimum (Left Ctrl)
		0x29, 0xE7, //   Usage Maximum (Right GUI)
		0x15, 0x00, //   Logical Minimum (0)
		0x25, 0x01, //   Logical Maximum (1)
		0x75, 0x01, //   Report Size (1)
		0x95, 0x08, //   Report Count (8)
		0x81, 0x02, //   Input (Data,Var,Abs) modifiers
		0x95, 0x01, //   Report Count (1)
		0x75, 0x08, //   Report Size (8)
		0x81, 0x01, //   Input (Cnst) reserved
		0x95, 0x06, //   Report Count (6)
		0x75, 0x08, //   Report Size (8)
		0x15, 0x00, //   Logical Minimum (0)
		0x25, 0x73, //   Logical Maximum (F24)
		0x05, 0x07, //   Usage Page (Keyboard)
		0x19, 0x00, //   Usage Minimum (0)
		0x29, 0x73, //   Usage Maximum
		0x81, 0x00, //   Input (Data,Ary,Abs)
		0xC0, // End Collection

		// Mouse collection
		0x05, 0x01, // Usage Page (Generic Desktop)
		0x09, 0x02, // Usage (Mouse)
		0xA1, 0x01, // Collection (Application)
		0x85, 0x02, //   Report ID (2)
		0x09, 0x01, //   Usage (Pointer)
		0xA1, 0x00, //   Collection (Physical)
		0x05, 0x09, //     Usage Page (Button)
		0x19, 0x01, //     Usage Minimum (1)
		0x29, 0x05, //     Usage Maximum (5)
		0x15, 0x00, //     Logical Minimum (0)
		0x25, 0x01, //     Logical Maximum (1)
		0x95, 0x05, //     Report Count (5)
		0x75, 0x01, //     Report Size (1)
		0x81, 0x02, //     Input (Data,Var,Abs)
		0x95, 0x01, //     Report Count (1)
		0x75, 0x03, //     Report Size (3)
		0x81, 0x01, //     Input (Cnst)
		0x05, 0x01, //     Usage Page (Generic Desktop)
		0x09, 0x30, //     Usage (X)
		0x09, 0x31, //     Usage (Y)
		0x09, 0x38, //     Usage (Wheel)
		0x15, 0x81, //     Logical Minimum (-127)
		0x25, 0x7F, //     Logical Maximum (127)
		0x75, 0x08, //     Report Size (8)
		0x95, 0x03, //     Report Count (3)
		0x81, 0x06, //     Input (Data,Var,Rel)
		0xC0, //   End Collection
		0xC0, // End Collection
	}
}

func DescriptorValid(desc []byte) bool {
	if len(desc) < 16 {
		return false
	}
	depth := 0
	i := 0
	for i < len(desc) {
		b := desc[i]
		size := int(b & 0x03)
		if size == 3 {
			size = 4
		}
		if b&0xFC == 0xA0 { // Collection
			depth++
		}
		if b == 0xC0 {
			depth--
			if depth < 0 {
				return false
			}
		}
		i += 1 + size
	}
	return depth == 0 && i == len(desc)
}
