import QtQuick
import Quickshell
import qs.Commons
import qs.Ui

// myna's icon in Omarchy's bar, put there by myna (Settings → Bar icon).
// A click drops down its panel under the bar, where the click was; a right
// click opens its window. myna is started if it isn't running.
BarWidget {
  id: root
  moduleName: "usemyna.myna"

  // written in by myna when it puts the widget here
  readonly property string myna: "__MYNA__"

  implicitWidth: button.implicitWidth
  implicitHeight: button.implicitHeight

  BarIconButton {
    id: button
    anchors.fill: parent
    bar: root.bar
    text: String.fromCodePoint(0xf15c6) // nf-md-bird
    tooltipText: "myna"
    onPressed: function(b) {
      Quickshell.execDetached(b === Qt.RightButton ? [root.myna, "app"] : [root.myna, "panel"])
    }
  }
}
