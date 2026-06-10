import usb_hid
import storage

# Disable CIRCUITPY USB drive visibility to host (keeps drive accessible internally)
# Comment this out if you need to edit files on the device without a second connection
storage.disable_usb_drive()

# Enable HID keyboard only — no serial, no MIDI
usb_hid.enable((usb_hid.Device.KEYBOARD,))
