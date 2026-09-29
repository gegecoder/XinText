/**
 * 构建配置生成器：
 * 读取 app.config.json，把 template/ 下 *.template 中的 ${key.path} 表达式
 * 替换为对应值，生成到构建目录（build/config.yml、build/windows/info.json）。
 * 在 wails3 build / package 之前自动执行（见 Taskfile.yml 的 generate:build:config）。
 */
import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..')
const configPath = path.join(root, 'app.config.json')
const config = JSON.parse(fs.readFileSync(configPath, 'utf8'))

// [模板文件, 生成目标, 目标是否为 JSON]
const jobs = [
  ['template/info.json.template', 'build/windows/info.json', true],
  ['template/config.yml.template', 'build/config.yml', false],
]

function getValue(keyPath) {
  return keyPath.split('.').reduce((o, k) => (o == null ? undefined : o[k]), config)
}

function render(template, isJson) {
  return template.replace(/\$\{([\w.]+)\}/g, (match, key) => {
    const value = getValue(key)
    if (value === undefined || value === null) {
      throw new Error(`app.config.json 缺少模板变量: ${key}`)
    }
    const str = String(value)
    // JSON 目标：按 JSON 字符串规则转义（去掉首尾引号），防止引号/反斜杠破坏结构
    return isJson ? JSON.stringify(str).slice(1, -1) : str
  })
}

for (const [tplRel, destRel, isJson] of jobs) {
  const tplPath = path.join(root, tplRel)
  const destPath = path.join(root, destRel)
  const output = render(fs.readFileSync(tplPath, 'utf8'), isJson)
  fs.mkdirSync(path.dirname(destPath), { recursive: true })
  fs.writeFileSync(destPath, output, 'utf8')
  console.log(`generated ${destRel} <- ${tplRel}`)
}
