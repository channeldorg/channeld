/**
 * 自动化测试脚本 - 使用 Puppeteer 测试 Inspector 前端
 * 
 * 功能：
 * - 自动启动浏览器并访问前端应用
 * - 捕获控制台日志
 * - 验证 WebSocket 连接
 * - 验证频道列表加载
 * - 生成测试报告
 */

import puppeteer from 'puppeteer'
import fs from 'fs'
import path from 'path'
import { fileURLToPath } from 'url'

const __filename = fileURLToPath(import.meta.url)
const __dirname = path.dirname(__filename)

// 配置
const CONFIG = {
  frontendUrl: 'http://localhost:5173',
  backendUrl: 'ws://localhost:80',
  timeout: 30000, // 30秒超时
  waitForChannels: 10000, // 等待频道列表加载的时间
}

// 测试结果
const testResults = {
  startTime: new Date(),
  logs: [],
  errors: [],
  warnings: [],
  testSteps: [],
  passed: 0,
  failed: 0,
}

// 日志收集器
function collectLog(type, message, args = []) {
  const logEntry = {
    timestamp: new Date().toISOString(),
    type,
    message,
    args: args.map(arg => {
      if (typeof arg === 'object') {
        try {
          return JSON.stringify(arg, null, 2)
        } catch {
          return String(arg)
        }
      }
      return String(arg)
    }),
  }
  
  testResults.logs.push(logEntry)
  
  if (type === 'error') {
    testResults.errors.push(logEntry)
  } else if (type === 'warning') {
    testResults.warnings.push(logEntry)
  }
  
  // 同时输出到控制台
  console.log(`[${type.toUpperCase()}]`, message, ...args)
}

// 测试步骤记录
function recordStep(name, status, details = '') {
  const step = {
    name,
    status,
    details,
    timestamp: new Date().toISOString(),
  }
  testResults.testSteps.push(step)
  
  if (status === 'passed') {
    testResults.passed++
    console.log(`✓ ${name}`)
  } else {
    testResults.failed++
    console.log(`✗ ${name}: ${details}`)
  }
}

async function runTests() {
  console.log('=== Inspector 前端自动化测试 ===\n')
  console.log(`前端地址: ${CONFIG.frontendUrl}`)
  console.log(`后端地址: ${CONFIG.backendUrl}\n`)
  
  let browser = null
  
  try {
    // 启动浏览器
    recordStep('启动浏览器', 'running')
    const headless = process.env.HEADLESS === 'true'
    browser = await puppeteer.launch({
      headless: headless, // 从环境变量读取
      devtools: !headless, // 无头模式时不打开开发者工具
      args: [
        '--disable-web-security',
        '--disable-features=IsolateOrigins,site-per-process',
      ],
    })
    recordStep('启动浏览器', 'passed')
    
    const page = await browser.newPage()
    
    // 设置视口大小
    await page.setViewport({ width: 1920, height: 1080 })
    
    // 监听控制台日志
    page.on('console', async (msg) => {
      const type = msg.type()
      const text = msg.text()
      const args = await Promise.all(
        msg.args().map(async (arg) => {
          try {
            const jsonValue = await arg.jsonValue()
            return jsonValue
          } catch {
            try {
              return await arg.toString()
            } catch {
              return String(arg)
            }
          }
        })
      )
      
      collectLog(type, text, args)
    })
    
    // 监听页面错误
    page.on('pageerror', (error) => {
      collectLog('error', `页面错误: ${error.message}`, [error.stack])
    })
    
    // 监听请求失败
    page.on('requestfailed', (request) => {
      collectLog('error', `请求失败: ${request.url()}`, [request.failure()?.errorText])
    })
    
    // 导航到前端应用
    recordStep('加载前端页面', 'running')
    await page.goto(CONFIG.frontendUrl, {
      waitUntil: 'networkidle2',
      timeout: CONFIG.timeout,
    })
    recordStep('加载前端页面', 'passed')
    
    // 等待页面加载完成
    await new Promise(resolve => setTimeout(resolve, 2000))
    
    // 检查连接状态
    recordStep('检查 WebSocket 连接', 'running')
    
    // 等待连接状态元素出现
    try {
      await page.waitForSelector('.connection-status', { timeout: 5000 })
    } catch (e) {
      recordStep('检查 WebSocket 连接', 'failed', '连接状态元素未出现')
    }
    
    const connectionStatus = await page.evaluate(() => {
      // 检查是否有连接状态元素
      const statusEl = document.querySelector('.connection-status')
      if (statusEl) {
        return {
          text: statusEl.textContent,
          hasConnected: statusEl.classList.contains('connected'),
        }
      }
      return null
    })
    
    if (connectionStatus) {
      console.log('连接状态:', connectionStatus)
      if (connectionStatus.hasConnected) {
        recordStep('检查 WebSocket 连接', 'passed', '已连接')
      } else {
        recordStep('检查 WebSocket 连接', 'running', `状态: ${connectionStatus.text}`)
        // 等待连接建立
        try {
          await page.waitForFunction(
            () => {
              const statusEl = document.querySelector('.connection-status.connected')
              return statusEl !== null
            },
            { timeout: CONFIG.waitForChannels }
          )
          recordStep('检查 WebSocket 连接', 'passed', '连接已建立')
        } catch (e) {
          recordStep('检查 WebSocket 连接', 'failed', `等待连接超时: ${e.message}`)
        }
      }
    } else {
      recordStep('检查 WebSocket 连接', 'failed', '未找到连接状态元素')
    }
    
    // 等待频道列表加载
    recordStep('等待频道列表加载', 'running')
    
    // 等待一段时间让频道列表加载
    console.log(`等待 ${CONFIG.waitForChannels}ms 让频道列表加载...`)
    await new Promise(resolve => setTimeout(resolve, CONFIG.waitForChannels))
    
    // 检查日志中是否有频道信息
    const channelLogs = testResults.logs.filter(log => {
      const msg = log.message.toLowerCase()
      return msg.includes('channels') || 
             msg.includes('channel') ||
             (msg.includes('received') && msg.includes('channel')) ||
             msg.includes('inspector_get_channels')
    })
    
    if (channelLogs.length > 0) {
      console.log('\n找到频道相关日志:')
      channelLogs.slice(-10).forEach(log => { // 只显示最后10条
        console.log(`  [${log.type}] ${log.message}`)
        if (log.args && log.args.length > 0) {
          log.args.forEach(arg => {
            const argStr = typeof arg === 'string' ? arg : JSON.stringify(arg)
            if (argStr.length < 200) { // 只显示较短的参数
              console.log(`    ${argStr}`)
            }
          })
        }
      })
    }
    
    // 检查是否有全局频道（channelId: 0）
    const hasGlobalChannel = testResults.logs.some(log => {
      const msg = log.message.toLowerCase()
      const argsStr = log.args.map(a => {
        try {
          return typeof a === 'string' ? a : JSON.stringify(a)
        } catch {
          return String(a)
        }
      }).join(' ').toLowerCase()
      
      return msg.includes('channelid: 0') ||
             msg.includes('channelid":0') ||
             msg.includes('channelid": 0') ||
             argsStr.includes('channelid":0') ||
             argsStr.includes('channelid": 0') ||
             msg.includes('global') && msg.includes('channel')
    })
    
    // 检查频道响应消息 - 更广泛的匹配
    const channelsResponseLogs = testResults.logs.filter(log => {
      const msg = log.message.toLowerCase()
      const allText = (log.message + ' ' + JSON.stringify(log.args)).toLowerCase()
      
      return msg.includes('received channels response') ||
             msg.includes('decoded message type 91') ||
             msg.includes('decoded message') && allText.includes('91') ||
             msg.includes('debug_get_channels_result') ||
             msg.includes('handling rpc response') && allText.includes('91') ||
             (allText.includes('channels') && allText.includes('total')) ||
             (allText.includes('channelid') && allText.includes('channeltype')) ||
             (allText.includes('debug') && allText.includes('channels') && allText.includes('response'))
    })
    
    let foundChannels = false
    let channelsCount = 0
    
    // 检查是否有频道响应
    if (channelsResponseLogs.length > 0) {
      foundChannels = true
      console.log(`\n找到 ${channelsResponseLogs.length} 条频道响应相关日志`)
      
      // 尝试从日志中提取频道数量
      const allLogsText = JSON.stringify(testResults.logs).toLowerCase()
      const totalMatch = allLogsText.match(/total["\s:]+(\d+)/i)
      const channelsMatch = allLogsText.match(/(\d+)\s+channels?/i)
      const lengthMatch = allLogsText.match(/channels.*length["\s:]+(\d+)/i)
      
      if (totalMatch) {
        channelsCount = parseInt(totalMatch[1])
      } else if (channelsMatch) {
        channelsCount = parseInt(channelsMatch[1])
      } else if (lengthMatch) {
        channelsCount = parseInt(lengthMatch[1])
      }
      
      console.log(`频道数量: ${channelsCount || '未知'}`)
      
      // 显示响应日志详情
      channelsResponseLogs.forEach(log => {
        console.log(`  [${log.type}] ${log.message}`)
        if (log.args && log.args.length > 0) {
          log.args.forEach((arg, idx) => {
            try {
              const argStr = typeof arg === 'object' ? JSON.stringify(arg, null, 2) : String(arg)
              if (argStr.length < 500) {
                console.log(`    arg[${idx}]: ${argStr}`)
              }
            } catch (e) {
              console.log(`    arg[${idx}]: [无法序列化]`)
            }
          })
        }
      })
    } else {
      console.log('\n未找到频道响应日志')
      console.log('检查是否有响应消息...')
      
      // 检查所有包含消息类型的日志
      const rpcLogs = testResults.logs.filter(log => 
        log.message.includes('RPC response') || 
        log.message.includes('Handling RPC') ||
        log.message.includes('msgType')
      )
      
      if (rpcLogs.length > 0) {
        console.log(`找到 ${rpcLogs.length} 条 RPC 相关日志:`)
        rpcLogs.slice(-5).forEach(log => {
          console.log(`  [${log.type}] ${log.message}`)
        })
      }
    }
    
    if (hasGlobalChannel || foundChannels) {
      recordStep('等待频道列表加载', 'passed', 
        foundChannels ? `找到 ${channelsCount} 个频道` : '从日志中找到频道信息（包括全局频道）')
    } else {
      recordStep('等待频道列表加载', 'failed', '未找到频道列表，请检查后端是否正常运行')
      console.log('\n提示：')
      console.log('1. 确保后端服务器正在运行 (channeld.exe)')
      console.log('2. 确保后端 Inspector 监听器在端口 80 上运行')
      console.log('3. 检查 WebSocket 连接是否成功建立')
    }
    
    // 截图（可选）
    const screenshotPath = path.join(__dirname, 'test-screenshot.png')
    await page.screenshot({ path: screenshotPath, fullPage: true })
    console.log(`\n截图已保存: ${screenshotPath}`)
    
    // 等待一段时间以收集更多日志
    console.log('\n等待 3 秒以收集更多日志...')
    await new Promise(resolve => setTimeout(resolve, 3000))
    
    // 最后检查一次连接状态
    const finalStatus = await page.evaluate(() => {
      const statusEl = document.querySelector('.connection-status')
      return statusEl ? {
        text: statusEl.textContent,
        connected: statusEl.classList.contains('connected'),
      } : null
    })
    
    if (finalStatus) {
      console.log(`最终连接状态: ${finalStatus.text} (${finalStatus.connected ? '已连接' : '未连接'})`)
    }
    
  } catch (error) {
    collectLog('error', `测试执行错误: ${error.message}`, [error.stack])
    recordStep('测试执行', 'failed', error.message)
  } finally {
    if (browser) {
      await browser.close()
    }
  }
  
  // 生成测试报告
  generateReport()
}

function generateReport() {
  testResults.endTime = new Date()
  testResults.duration = testResults.endTime - testResults.startTime
  
  console.log('\n=== 测试报告 ===\n')
  console.log(`总耗时: ${(testResults.duration / 1000).toFixed(2)} 秒`)
  console.log(`通过: ${testResults.passed}`)
  console.log(`失败: ${testResults.failed}`)
  console.log(`总日志数: ${testResults.logs.length}`)
  console.log(`错误数: ${testResults.errors.length}`)
  console.log(`警告数: ${testResults.warnings.length}`)
  
  // 保存详细报告到文件
  const reportPath = path.join(__dirname, 'test-report.json')
  fs.writeFileSync(reportPath, JSON.stringify(testResults, null, 2))
  console.log(`\n详细报告已保存: ${reportPath}`)
  
  // 生成 HTML 报告
  generateHTMLReport()
  
  // 输出关键日志
  if (testResults.errors.length > 0) {
    console.log('\n=== 错误日志 ===')
    testResults.errors.forEach(error => {
      console.log(`[${error.timestamp}] ${error.message}`)
      if (error.args.length > 0) {
        error.args.forEach(arg => console.log(`  ${arg}`))
      }
    })
  }
  
  // 输出频道相关日志
  const channelLogs = testResults.logs.filter(log => 
    log.message.toLowerCase().includes('channel') ||
    log.message.includes('INSPECTOR_GET_CHANNELS')
  )
  
  if (channelLogs.length > 0) {
    console.log('\n=== 频道相关日志 ===')
    channelLogs.forEach(log => {
      console.log(`[${log.timestamp}] [${log.type}] ${log.message}`)
    })
  }
}

function generateHTMLReport() {
  const html = `
<!DOCTYPE html>
<html>
<head>
  <meta charset="UTF-8">
  <title>Inspector 前端测试报告</title>
  <style>
    body { font-family: Arial, sans-serif; margin: 20px; background: #f5f5f5; }
    .container { max-width: 1200px; margin: 0 auto; background: white; padding: 20px; border-radius: 8px; }
    h1 { color: #333; }
    .summary { display: flex; gap: 20px; margin: 20px 0; }
    .stat { background: #f0f0f0; padding: 15px; border-radius: 5px; flex: 1; }
    .stat h3 { margin: 0 0 10px 0; color: #666; }
    .stat .value { font-size: 2em; font-weight: bold; }
    .passed { color: #4caf50; }
    .failed { color: #f44336; }
    .log-entry { padding: 10px; margin: 5px 0; border-left: 4px solid #ddd; background: #fafafa; }
    .log-entry.error { border-color: #f44336; }
    .log-entry.warning { border-color: #ff9800; }
    .log-entry.info { border-color: #2196f3; }
    .timestamp { color: #999; font-size: 0.9em; }
    pre { background: #f5f5f5; padding: 10px; border-radius: 4px; overflow-x: auto; }
  </style>
</head>
<body>
  <div class="container">
    <h1>Inspector 前端测试报告</h1>
    <div class="summary">
      <div class="stat">
        <h3>通过</h3>
        <div class="value passed">${testResults.passed}</div>
      </div>
      <div class="stat">
        <h3>失败</h3>
        <div class="value failed">${testResults.failed}</div>
      </div>
      <div class="stat">
        <h3>总日志</h3>
        <div class="value">${testResults.logs.length}</div>
      </div>
      <div class="stat">
        <h3>错误</h3>
        <div class="value failed">${testResults.errors.length}</div>
      </div>
    </div>
    
    <h2>测试步骤</h2>
    <ul>
      ${testResults.testSteps.map(step => `
        <li>
          <strong>${step.name}</strong>: 
          <span class="${step.status}">${step.status}</span>
          ${step.details ? ` - ${step.details}` : ''}
        </li>
      `).join('')}
    </ul>
    
    <h2>错误日志</h2>
    ${testResults.errors.length > 0 ? testResults.errors.map(error => `
      <div class="log-entry error">
        <div class="timestamp">${error.timestamp}</div>
        <div>${error.message}</div>
        ${error.args.length > 0 ? `<pre>${error.args.join('\n')}</pre>` : ''}
      </div>
    `).join('') : '<p>无错误</p>'}
    
    <h2>所有日志</h2>
    <details>
      <summary>展开查看所有日志 (${testResults.logs.length} 条)</summary>
      ${testResults.logs.map(log => `
        <div class="log-entry ${log.type}">
          <div class="timestamp">${log.timestamp}</div>
          <div>[${log.type}] ${log.message}</div>
          ${log.args.length > 0 ? `<pre>${log.args.join('\n')}</pre>` : ''}
        </div>
      `).join('')}
    </details>
  </div>
</body>
</html>
  `
  
  const htmlPath = path.join(__dirname, 'test-report.html')
  fs.writeFileSync(htmlPath, html)
  console.log(`HTML 报告已保存: ${htmlPath}`)
}

// 运行测试
runTests().catch(error => {
  console.error('测试执行失败:', error)
  process.exit(1)
})

export { runTests }
