import QtQuick
import Quickshell
import qs.Commons
import qs.Ui

// owl's icon in Omarchy's bar, put there by owl (Settings → Bar icon).
// A click drops down its panel under the bar, where the click was; a right
// click opens its window. owl is started if it isn't running.
BarWidget {
  id: root
  moduleName: "useowl.owl"

  // written in by owl when it puts the widget here
  readonly property string owl: "__OWL__"

  implicitWidth: button.implicitWidth
  implicitHeight: button.implicitHeight

  BarIconButton {
    id: button
    anchors.fill: parent
    bar: root.bar
    text: String.fromCodePoint(0xf15c6) // nf-md-bird
    tooltipText: "owl"
    onPressed: function(b) {
      Quickshell.execDetached(b === Qt.RightButton ? [root.owl, "app"] : [root.owl, "panel"])
    }
  }
}
