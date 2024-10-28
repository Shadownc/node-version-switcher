import { useState, useEffect } from 'react';
import './App.css';
import {
    SwitchNodeVersion,
    GetAvailableNodeVersions,
    GetInstalledNodeVersions,
    InstallNodeVersion
} from '../wailsjs/go/main/App';

// 改进版 compareVersions，确保按主、次、小版本的降序排列
const compareVersions = (a, b) => {
    const parseVersion = (version) => version.split('.').map((num) => parseInt(num, 10));
    const [aMajor, aMinor, aPatch] = parseVersion(a.Version);
    const [bMajor, bMinor, bPatch] = parseVersion(b.Version);

    if (aMajor !== bMajor) return bMajor - aMajor;
    if (aMinor !== bMinor) return bMinor - aMinor;
    return bPatch - aPatch;
};

function App() {
    const [activeTab, setActiveTab] = useState('versions');
    const [availableVersions, setAvailableVersions] = useState([]);
    const [installedVersions, setInstalledVersions] = useState([]);
    const [result, setResult] = useState('');
    const [maxHeight, setMaxHeight] = useState(window.innerHeight - 220); // 初始化 maxHeight

    // 动态计算 maxHeight，当窗口大小变化时更新
    useEffect(() => {
        const updateMaxHeight = () => setMaxHeight(window.innerHeight - 220);
        window.addEventListener('resize', updateMaxHeight);
        return () => window.removeEventListener('resize', updateMaxHeight);
    }, []);

    // 获取最新已安装版本列表
    const fetchInstalledVersions = async () => {
        try {
            const installed = await GetInstalledNodeVersions();
            setInstalledVersions(installed.sort(compareVersions));
        } catch (error) {
            setResult('Error fetching installed versions');
        }
    };

    // 获取所有版本信息
    useEffect(() => {
        const fetchVersions = async () => {
            try {
                const available = await GetAvailableNodeVersions();
                setAvailableVersions(available.sort(compareVersions));
                await fetchInstalledVersions();
            } catch (error) {
                setResult('Error fetching versions');
            }
        };
        fetchVersions();
    }, []);

    const handleInstallVersion = async (version) => {
        try {
            const response = await InstallNodeVersion(version);
            setResult(response);
            await fetchInstalledVersions();
        } catch (error) {
            setResult('Error installing version');
        }
    };

    const handleSwitchVersion = async (version) => {
        try {
            const response = await SwitchNodeVersion(version);
            setResult(response);
            await fetchInstalledVersions();
        } catch (error) {
            setResult('Error switching version');
        }
    };

    return (
        <div className="App min-h-screen bg-gray-900 text-white p-6 overflow-hidden">
            <h1 className="text-3xl font-bold mb-6 text-center">Node.js Version Manager</h1>

            <div className="tabs flex justify-center space-x-4 mb-6">
                <button
                    className={`px-4 py-2 ${activeTab === 'versions' ? 'bg-blue-500' : 'bg-gray-700'} rounded`}
                    onClick={() => setActiveTab('versions')}
                >
                    Versions
                </button>
                <button
                    className={`px-4 py-2 ${activeTab === 'installed' ? 'bg-blue-500' : 'bg-gray-700'} rounded`}
                    onClick={() => setActiveTab('installed')}
                >
                    Installed
                </button>
            </div>

            {activeTab === 'versions' && (
                <div
                    className="version-list bg-gray-800 p-4 rounded-lg custom-scrollbar"
                    style={{ maxHeight: `${maxHeight}px`, overflowY: 'auto' }}
                >
                    <h2 className="text-xl font-semibold mb-4">Available Versions</h2>
                    <table className="table-auto w-full text-left">
                        <thead>
                            <tr>
                                <th className="px-4 py-2">Version</th>
                                <th className="px-4 py-2">Status</th>
                                <th className="px-4 py-2">Operation</th>
                            </tr>
                        </thead>
                        <tbody>
                            {availableVersions.map((versionInfo, index) => (
                                <tr key={index} className="border-t border-gray-700">
                                    <td className="px-4 py-2">{versionInfo.Version}</td>
                                    <td className="px-4 py-2">
                                        {versionInfo.Status === "Installed" ? (
                                            <span className="text-green-500">Installed</span>
                                        ) : (
                                            <span className="text-red-500">Not Installed</span>
                                        )}
                                    </td>
                                    <td className="px-4 py-2">
                                        {versionInfo.Status === "Installed" ? (
                                            <button
                                                className="bg-gray-600 px-3 py-1 rounded text-gray-300"
                                                onClick={() => handleSwitchVersion(versionInfo.Version)}
                                            >
                                                Use
                                            </button>
                                        ) : (
                                            <button
                                                className="bg-blue-600 px-3 py-1 rounded text-white"
                                                onClick={() => handleInstallVersion(versionInfo.Version)}
                                            >
                                                Install
                                            </button>
                                        )}
                                    </td>
                                </tr>
                            ))}
                        </tbody>
                    </table>
                </div>
            )}

            {activeTab === 'installed' && (
                <div
                    className="installed-list bg-gray-800 p-4 rounded-lg custom-scrollbar"
                    style={{ maxHeight: `${maxHeight}px`, overflowY: 'auto' }}
                >
                    <h2 className="text-xl font-semibold mb-4">Installed Versions</h2>
                    <table className="table-auto w-full text-left">
                        <thead>
                            <tr>
                                <th className="px-4 py-2">Version</th>
                                <th className="px-4 py-2">Operation</th>
                            </tr>
                        </thead>
                        <tbody>
                            {installedVersions.map((versionData, index) => (
                                <tr key={index} className="border-t border-gray-700">
                                    <td className="px-4 py-2">
                                        {versionData.Version}{" "}
                                        {versionData.IsCurrent && (
                                            <span className="text-sm text-blue-400">(Current)</span>
                                        )}
                                    </td>
                                    <td className="px-4 py-2">
                                        {versionData.IsCurrent ? (
                                            <button
                                                className="bg-gray-600 px-3 py-1 rounded text-gray-300 cursor-not-allowed"
                                                disabled
                                            >
                                                Current
                                            </button>
                                        ) : (
                                            <>
                                                <button
                                                    className="bg-green-600 px-3 py-1 rounded text-white mr-2"
                                                    onClick={() => handleSwitchVersion(versionData.Version)}
                                                >
                                                    Switch
                                                </button>
                                                <button
                                                    className="bg-red-600 px-3 py-1 rounded text-white"
                                                    onClick={() => {/* 添加卸载功能 */ }}
                                                >
                                                    Uninstall
                                                </button>
                                            </>
                                        )}
                                    </td>
                                </tr>
                            ))}
                        </tbody>
                    </table>
                </div>
            )}

            <p className="text-sm text-gray-400 mt-4">{result}</p>
        </div>
    );
}

export default App;
