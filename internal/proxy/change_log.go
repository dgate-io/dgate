package proxy

import (
	"fmt"
	"time"

	"errors"

	"github.com/dgate-io/dgate/pkg/spec"
	"github.com/dgate-io/dgate/pkg/util/sliceutil"
	"github.com/dgraph-io/badger/v4"
	"github.com/mitchellh/mapstructure"
	"go.uber.org/zap"
)

// processChangeLog - processes a change log and applies the change to the proxy state
func (ps *ProxyState) processChangeLog(cl *spec.ChangeLog, reload, store bool) (err error) {
	if cl == nil {
		cl = &spec.ChangeLog{
			Cmd: spec.NoopCommand,
		}
	} else if !cl.Cmd.IsNoop() {
		switch cl.Cmd.Resource() {
		case spec.Namespaces:
			var item spec.Namespace
			item, err = decode[spec.Namespace](cl.Item)
			if err == nil {
				ps.logger.Debug("Processing namespace", zap.String("name", item.Name))
				err = ps.processNamespace(&item, cl)
			}
		case spec.Services:
			var item spec.Service
			item, err = decode[spec.Service](cl.Item)
			if err == nil {
				ps.logger.Debug("Processing service", zap.String("name", item.Name))
				err = ps.processService(&item, cl)
			}
		case spec.Routes:
			var item spec.Route
			item, err = decode[spec.Route](cl.Item)
			if err == nil {
				ps.logger.Debug("Processing route", zap.String("name", item.Name))
				err = ps.processRoute(&item, cl)
			}
		case spec.Modules:
			var item spec.Module
			item, err = decode[spec.Module](cl.Item)
			if err == nil {
				ps.logger.Debug("Processing module", zap.String("name", item.Name))
				err = ps.processModule(&item, cl)
			}
		case spec.Domains:
			var item spec.Domain
			item, err = decode[spec.Domain](cl.Item)
			if err == nil {
				ps.logger.Debug("Processing domain", zap.String("name", item.Name))
				err = ps.processDomain(&item, cl)
			}
		case spec.Collections:
			var item spec.Collection
			item, err = decode[spec.Collection](cl.Item)
			if err == nil {
				ps.logger.Debug("Processing collection", zap.String("name", item.Name))
				err = ps.processCollection(&item, cl)
			}
		case spec.Documents:
			var item spec.Document
			item, err = decode[spec.Document](cl.Item)
			if err == nil {
				ps.logger.Debug("Processing document", zap.String("id", item.ID))
				err = ps.processDocument(&item, cl)
			}
		case spec.Secrets:
			var item spec.Secret
			item, err = decode[spec.Secret](cl.Item)
			if err == nil {
				ps.logger.Debug("Processing secret", zap.String("name", item.Name))
				err = ps.processSecret(&item, cl)
			}
		default:
			err = fmt.Errorf("unknown command: %s", cl.Cmd)
		}
		if err != nil {
			ps.logger.Error("decoding or processing change log", zap.Error(err))
			return
		}
	}
	if reload {
		if cl.Cmd.Resource().IsRelatedTo(spec.Routes) || cl.Cmd.IsNoop() {
			ps.logger.Debug("Registering change log", zap.Stringer("cmd", cl.Cmd))
			err = ps.reconfigureState(false, cl)
			if err != nil {
				ps.logger.Error("Error registering change log", zap.Error(err))
				return
			}
			// update change log hash only when the change is successfully applied
			//   even if the change is a noop, we still need to update the hash
			changeHash, err := HashAny(ps.changeHash, cl)
			if err != nil {
				if !ps.config.Debug {
					return err
				}
				ps.logger.Error("error updating change log hash", zap.Error(err))
			} else {
				ps.changeHash = changeHash
			}
		}
	}
	if store {
		if err = ps.store.StoreChangeLog(cl); err != nil {
			// TODO: find a way to revert the change and reload the state
			// TODO: OR add flag in config to ignore storage errors
			ps.logger.Error("Error storing change log", zap.Error(err))
			return
		}
	}

	return nil
}

func decode[T any](input any) (T, error) {
	var output T
	cfg := &mapstructure.DecoderConfig{
		Metadata: nil,
		Result:   &output,
		TagName:  "json",
		DecodeHook: mapstructure.ComposeDecodeHookFunc(
			mapstructure.StringToTimeHookFunc(time.RFC3339),
			mapstructure.StringToTimeDurationHookFunc(),
		),
	}
	if dec, err := mapstructure.NewDecoder(cfg); err != nil {
		return output, err
	} else if err = dec.Decode(input); err != nil {
		return output, err
	}
	return output, nil
}

func (ps *ProxyState) processNamespace(ns *spec.Namespace, cl *spec.ChangeLog) error {
	switch cl.Cmd.Action() {
	case spec.Add:
		ps.rm.AddNamespace(ns)
	case spec.Delete:
		return ps.rm.RemoveNamespace(ns.Name)
	default:
		return fmt.Errorf("unknown command: %s", cl.Cmd)
	}
	return nil
}

func (ps *ProxyState) processService(svc *spec.Service, cl *spec.ChangeLog) (err error) {
	if svc.NamespaceName == "" {
		svc.NamespaceName = cl.Namespace
	}
	switch cl.Cmd.Action() {
	case spec.Add:
		_, err = ps.rm.AddService(svc)
	case spec.Delete:
		err = ps.rm.RemoveService(svc.Name, svc.NamespaceName)
	default:
		err = fmt.Errorf("unknown command: %s", cl.Cmd)
	}
	return err
}

func (ps *ProxyState) processRoute(rt *spec.Route, cl *spec.ChangeLog) (err error) {
	if rt.NamespaceName == "" {
		rt.NamespaceName = cl.Namespace
	}
	switch cl.Cmd.Action() {
	case spec.Add:
		_, err = ps.rm.AddRoute(rt)
	case spec.Delete:
		err = ps.rm.RemoveRoute(rt.Name, rt.NamespaceName)
	default:
		err = fmt.Errorf("unknown command: %s", cl.Cmd)
	}
	return err
}

func (ps *ProxyState) processModule(mod *spec.Module, cl *spec.ChangeLog) (err error) {
	if mod.NamespaceName == "" {
		mod.NamespaceName = cl.Namespace
	}
	switch cl.Cmd.Action() {
	case spec.Add:
		_, err = ps.rm.AddModule(mod)
	case spec.Delete:
		err = ps.rm.RemoveModule(mod.Name, mod.NamespaceName)
	default:
		err = fmt.Errorf("unknown command: %s", cl.Cmd)
	}
	return err
}

func (ps *ProxyState) processDomain(dom *spec.Domain, cl *spec.ChangeLog) (err error) {
	if dom.NamespaceName == "" {
		dom.NamespaceName = cl.Namespace
	}
	switch cl.Cmd.Action() {
	case spec.Add:
		_, err = ps.rm.AddDomain(dom)
	case spec.Delete:
		err = ps.rm.RemoveDomain(dom.Name, dom.NamespaceName)
	default:
		err = fmt.Errorf("unknown command: %s", cl.Cmd)
	}
	return err
}

func (ps *ProxyState) processCollection(col *spec.Collection, cl *spec.ChangeLog) (err error) {
	if col.NamespaceName == "" {
		col.NamespaceName = cl.Namespace
	}
	switch cl.Cmd.Action() {
	case spec.Add:
		_, err = ps.rm.AddCollection(col)
	case spec.Delete:
		err = ps.rm.RemoveCollection(col.Name, col.NamespaceName)
	default:
		err = fmt.Errorf("unknown command: %s", cl.Cmd)
	}
	return err
}

func (ps *ProxyState) processDocument(doc *spec.Document, cl *spec.ChangeLog) (err error) {
	if doc.NamespaceName == "" {
		doc.NamespaceName = cl.Namespace
	}
	switch cl.Cmd.Action() {
	case spec.Add:
		err = ps.store.StoreDocument(doc)
	case spec.Delete:
		err = ps.store.DeleteDocument(doc.ID, doc.CollectionName, doc.NamespaceName)
	default:
		err = fmt.Errorf("unknown command: %s", cl.Cmd)
	}
	return err
}

func (ps *ProxyState) processSecret(scrt *spec.Secret, cl *spec.ChangeLog) (err error) {
	if scrt.NamespaceName == "" {
		scrt.NamespaceName = cl.Namespace
	}
	switch cl.Cmd.Action() {
	case spec.Add:
		_, err = ps.rm.AddSecret(scrt)
	case spec.Delete:
		err = ps.rm.RemoveSecret(scrt.Name, scrt.NamespaceName)
	default:
		err = fmt.Errorf("unknown command: %s", cl.Cmd)
	}
	return err
}

// applyChange - apply a change to the proxy state, returns a channel that will receive an error when the state has been updated
func (ps *ProxyState) applyChange(changeLog *spec.ChangeLog) <-chan error {
	done := make(chan error, 1)
	if changeLog == nil {
		changeLog = &spec.ChangeLog{
			Cmd: spec.NoopCommand,
		}
	}
	changeLog.SetErrorChan(done)
	if err := ps.processChangeLog(changeLog, true, true); err != nil {
		done <- err
	}
	return done
}

func (ps *ProxyState) restoreFromChangeLogs(directApply bool) error {
	// restore state change logs
	logs, err := ps.store.FetchChangeLogs()
	if err != nil {
		if err == badger.ErrKeyNotFound {
			ps.logger.Debug("no state change logs found in storage")
		} else {
			return errors.New("failed to get state change logs from storage: " + err.Error())
		}
	} else {
		ps.logger.Info("restoring state change logs from storage", zap.Int("count", len(logs)))
		// we might need to sort the change logs by timestamp
		for i, cl := range logs {
			ps.logger.Debug("restoring change log",
				zap.Int("index", i),
				zap.Stringer("changeLog", cl.Cmd),
			)
			err = ps.processChangeLog(cl, false, false)
			if err != nil {
				if ps.config.Debug {
					ps.logger.Error("error restorng from change logs", zap.Error(err))
					continue
				}
				return err
			}
		}
		if !directApply {
			if err = ps.processChangeLog(nil, true, false); err != nil {
				return err
			}
		} else {
			if err = ps.reconfigureState(false, nil); err != nil {
				return nil
			}
		}

		// TODO: optionally compact change logs through a flag in config?
		if len(logs) > 1 {
			removed, err := ps.compactChangeLogs(logs)
			if err != nil {
				ps.logger.Error("failed to compact state change logs", zap.Error(err))
				return err
			}
			if removed > 0 {
				ps.logger.Info("compacted change logs",
					zap.Int("removed", removed),
					zap.Int("total", len(logs)),
				)
			}
		}
	}
	return nil
}

func (ps *ProxyState) compactChangeLogs(logs []*spec.ChangeLog) (int, error) {
	removeList := compactChangeLogsRemoveList(ps.logger, sliceutil.SliceCopy(logs))
	removed, err := ps.store.DeleteChangeLogs(removeList)
	if err != nil {
		return removed, err
	}
	return removed, nil
}

/*
compactChangeLogsRemoveList - compacts a list of change logs by removing redundant logs.

compaction rules:
  - if an add command is followed by a delete command with matching keys, remove both commands
  - if an add command is followed by another add command with matching keys, remove the first add command
*/
func compactChangeLogsRemoveList(logger *zap.Logger, logs []*spec.ChangeLog) []*spec.ChangeLog {
	removeList := make([]*spec.ChangeLog, 0)
	iterations := 0
START:
	var prevLog *spec.ChangeLog
	// TODO: this can be extended by separating the logs into namespace groups and then compacting each group
	for i := 0; i < len(logs); i++ {
		iterations++
		curLog := logs[i]
		if prevLog != nil {
			if prevLog.Cmd.IsNoop() {
				removeList = append(removeList, prevLog)
				logs = append(logs[:i-1], logs[i:]...)
				goto START
			}

			commonResource := prevLog.Cmd.Resource() == curLog.Cmd.Resource()
			if prevLog.Cmd.Action() == spec.Add && curLog.Cmd.Action() == spec.Delete && commonResource {
				// Rule 1: if an add command is followed by a delete
				//   command with matching keys, remove both commands
				if prevLog.Name == curLog.Name && prevLog.Namespace == curLog.Namespace {
					removeList = append(removeList, prevLog, curLog)
					logs = append(logs[:i-1], logs[i+1:]...)
					goto START
				}
			}

			commonAction := prevLog.Cmd.Action() == curLog.Cmd.Action()
			if prevLog.Cmd.Action() == spec.Add && commonAction && commonResource {
				// Rule 2: if an add command is followed by another add
				//   command with matching keys, remove the first add command
				if prevLog.Name == curLog.Name && prevLog.Namespace == curLog.Namespace {
					removeList = append(removeList, prevLog)
					logs = append(logs[:i-1], logs[i:]...)
					goto START
				}
			}
		}
		prevLog = curLog
	}
	logger.Debug("compacted change logs", zap.Int("iterations", iterations))

	// remove duplicates from list
	removeList = sliceutil.SliceUnique(removeList, func(cl *spec.ChangeLog) string { return cl.ID })
	return removeList
}

// Function to check if there is a delete command between two logs with matching keys
// func hasDeleteBetween(logs []*spec.ChangeLog, start, end *spec.ChangeLog) bool {
// 	startIndex := -1
// 	endIndex := -1

// 	for i, log := range logs {
// 		if log.ID == start.ID {
// 			startIndex = i
// 		}
// 		if log.ID == end.ID {
// 			endIndex = i
// 		}
// 	}

// 	if startIndex == -1 || endIndex == -1 {
// 		return false
// 	}

// 	for i := startIndex + 1; i < endIndex; i++ {
// 		if logs[i].Cmd.IsDeleteCommand() {
// 			return true
// 		}
// 	}
// 	return false
// }
