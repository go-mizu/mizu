import { createReactBlockSpec } from '@blocknote/react'

export const DividerBlock = createReactBlockSpec(
  {
    type: 'divider',
    propSchema: {},
    content: 'none',
  },
  {
    render: () => {
      return (
        <div className="divider-block-wrapper">
          <hr className="divider-block" />
        </div>
      )
    },
  }
)
