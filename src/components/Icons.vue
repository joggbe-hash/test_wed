<script>
import { h } from 'vue'

const iconProps = {
  width: '20',
  height: '20',
  viewBox: '0 0 24 24',
  fill: 'none',
  stroke: 'currentColor',
  'stroke-width': '2',
}

export const OpenEye = {
  render() {
    return h('svg', iconProps, [
      h('path', { d: 'M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z' }),
      h('circle', { cx: '12', cy: '12', r: '3' }),
    ])
  },
}

export const ClosedEye = {
  render() {
    return h('svg', { ...iconProps, 'stroke-linecap': 'round' }, [
      h('path', { d: 'M3 10 C8 16, 16 16, 21 10' }),
      h('line', { x1: '12', y1: '14', x2: '12', y2: '18' }),
      h('line', { x1: '8', y1: '13', x2: '6', y2: '16' }),
      h('line', { x1: '16', y1: '13', x2: '18', y2: '16' }),
      h('line', { x1: '10', y1: '13.5', x2: '9', y2: '17' }),
      h('line', { x1: '14', y1: '13.5', x2: '15', y2: '17' }),
    ])
  },
}
</script>
