// Cynhyrchwyd y ffeil hon yn awtomatig. PEIDIWCH Â MODIWL
// This file is automatically generated. DO NOT EDIT
import {main} from '../models';

export function GetAvailableNodeVersions():Promise<Array<main.NodeVersionInfo>>;

export function GetInstalledNodeVersions():Promise<Array<main.NodeVersion>>;

export function InstallNodeVersion(arg1:string):Promise<string>;

export function SwitchNodeVersion(arg1:string):Promise<string>;
