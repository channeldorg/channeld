import { useState, useCallback } from 'react'
import { useInspectorStore } from '../stores/inspectorStore'
import './JsonTreeEditor.css'

interface JsonTreeEditorProps {
  data: any
  channelId: number
}

interface TreeNode {
  key: string
  value: any
  path: string
  isLeaf: boolean
  children?: TreeNode[]
}

export default function JsonTreeEditor({ data, channelId }: JsonTreeEditorProps) {
  const { updateChannelData } = useInspectorStore()
  const [expandedPaths, setExpandedPaths] = useState<Set<string>>(new Set())
  const [editingPath, setEditingPath] = useState<string | null>(null)
  const [editValue, setEditValue] = useState<string>('')

  const buildTree = useCallback((obj: any, path: string = ''): TreeNode[] => {
    if (obj === null || obj === undefined) {
      return [{
        key: 'null',
        value: null,
        path,
        isLeaf: true,
      }]
    }

    if (Array.isArray(obj)) {
      return obj.map((item, index) => {
        const itemPath = path ? `${path}[${index}]` : `[${index}]`
        if (typeof item === 'object' && item !== null) {
          return {
            key: `[${index}]`,
            value: item,
            path: itemPath,
            isLeaf: false,
            children: buildTree(item, itemPath),
          }
        }
        return {
          key: `[${index}]`,
          value: item,
          path: itemPath,
          isLeaf: true,
        }
      })
    }

    if (typeof obj === 'object') {
      return Object.entries(obj).map(([key, value]) => {
        const itemPath = path ? `${path}.${key}` : key
        if (typeof value === 'object' && value !== null && !Array.isArray(value)) {
          return {
            key,
            value,
            path: itemPath,
            isLeaf: false,
            children: buildTree(value, itemPath),
          }
        }
        if (Array.isArray(value)) {
          return {
            key,
            value,
            path: itemPath,
            isLeaf: false,
            children: buildTree(value, itemPath),
          }
        }
        return {
          key,
          value,
          path: itemPath,
          isLeaf: true,
        }
      })
    }

    return [{
      key: 'value',
      value: obj,
      path,
      isLeaf: true,
    }]
  }, [])

  const tree = buildTree(data)

  const toggleExpanded = (path: string) => {
    setExpandedPaths(prev => {
      const next = new Set(prev)
      if (next.has(path)) {
        next.delete(path)
      } else {
        next.add(path)
      }
      return next
    })
  }

  const startEdit = (node: TreeNode) => {
    if (node.isLeaf) {
      setEditingPath(node.path)
      setEditValue(String(node.value))
    }
  }

  const saveEdit = () => {
    if (editingPath) {
      const valueType = getValueType(editValue)
      updateChannelData(channelId, editingPath, editValue, valueType)
      setEditingPath(null)
      setEditValue('')
    }
  }

  const cancelEdit = () => {
    setEditingPath(null)
    setEditValue('')
  }

  const getValueType = (value: string): string => {
    if (value === 'null' || value === '') return 'null'
    if (value === 'true' || value === 'false') return 'boolean'
    if (!isNaN(Number(value)) && value.trim() !== '') return 'number'
    if (value.startsWith('{') || value.startsWith('[')) return 'object'
    return 'string'
  }

  const renderNode = (node: TreeNode, depth: number = 0): JSX.Element => {
    const isExpanded = expandedPaths.has(node.path)
    const isEditing = editingPath === node.path

    return (
      <div key={node.path} className="tree-node" style={{ paddingLeft: `${depth * 16}px` }}>
        <div className="tree-node-header">
          {!node.isLeaf && (
            <button
              className="expand-button"
              onClick={() => toggleExpanded(node.path)}
            >
              {isExpanded ? '▼' : '▶'}
            </button>
          )}
          <span className="tree-node-key">{node.key}:</span>
          {node.isLeaf ? (
            isEditing ? (
              <div className="edit-controls">
                <input
                  type="text"
                  value={editValue}
                  onChange={(e) => setEditValue(e.target.value)}
                  onKeyDown={(e) => {
                    if (e.key === 'Enter') saveEdit()
                    if (e.key === 'Escape') cancelEdit()
                  }}
                  autoFocus
                  className="edit-input"
                />
                <button onClick={saveEdit} className="save-button">Save</button>
                <button onClick={cancelEdit} className="cancel-button">Cancel</button>
              </div>
            ) : (
              <span
                className="tree-node-value"
                onClick={() => startEdit(node)}
              >
                {formatValue(node.value)}
              </span>
            )
          ) : (
            <span className="tree-node-type">
              {Array.isArray(node.value) ? 'Array' : 'Object'}
            </span>
          )}
        </div>
        {!node.isLeaf && isExpanded && node.children && (
          <div className="tree-node-children">
            {node.children.map(child => renderNode(child, depth + 1))}
          </div>
        )}
      </div>
    )
  }

  const formatValue = (value: any): string => {
    if (value === null) return 'null'
    if (typeof value === 'string') return `"${value}"`
    return String(value)
  }

  return (
    <div className="json-tree-editor">
      <div className="json-tree">
        {tree.map(node => renderNode(node))}
      </div>
    </div>
  )
}
