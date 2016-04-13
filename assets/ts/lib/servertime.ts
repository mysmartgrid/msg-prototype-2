import {now} from './utils';
import {ServerTimeHandler, Socket} from './msg2socket';



const OffsetCount = 10;

export class ServerTime {
    private _offsets : number[];
    private _averageOffset : number;

    constructor(private _socket : Socket) {
        console.log("New ServerTime");
        this._averageOffset = 0;
        this._offsets = [];

        _socket.onServerTime((servertime) => this._updateOffsets(servertime));
    }


    private _updateOffsets(servertime : number) {
        if(this._offsets.length >= OffsetCount) {
            this._offsets.shift();
        }
        this._offsets.push(now() - servertime);

        this._averageOffset = 0;
        for(var offset of this._offsets) {
            this._averageOffset += offset / OffsetCount;
        }

        console.log("Timeoffset:", this._averageOffset);
    }


    public getOffset() : number {
        return this._averageOffset;
    }

    public now() : number {
        return now() - this._averageOffset;
    }
}


export const ServerTimeFactory =  ["WSUserClient", (socket : Socket) => new ServerTime(socket)];
