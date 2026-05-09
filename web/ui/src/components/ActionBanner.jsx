import { CheckCircle, ArrowRight, PauseCircle, AlertCircle, RotateCcw } from 'lucide-react'

const iconMap = {
  exit: CheckCircle,
  transfer: ArrowRight,
  interrupted: PauseCircle,
  break_loop: RotateCcw,
  error: AlertCircle,
}

const labelMap = {
  exit: 'Agent 执行完成',
  transfer: '转移中',
  interrupted: '等待外部输入',
  break_loop: '循环中断',
}

export default function ActionBanner({ message }) {
  const { actionType, content } = message
  const Icon = iconMap[actionType] || AlertCircle
  const label = labelMap[actionType] || actionType

  let cls = 'action-banner'
  if (actionType === 'exit') cls += ' exit'
  else if (actionType === 'transfer') cls += ' transfer'
  else if (actionType === 'interrupted') cls += ' interrupted'
  else cls += ' error'

  return (
    <div className={cls}>
      <Icon size={16} />
      <span>
        {label}
        {content && actionType === 'transfer' && `: ${content}`}
        {content && actionType === 'interrupted' && `: ${content}`}
      </span>
    </div>
  )
}
