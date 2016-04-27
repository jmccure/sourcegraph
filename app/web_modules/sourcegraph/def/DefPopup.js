// @flow weak

import React from "react";
import {Link} from "react-router";
import CSSModules from "react-css-modules";
import Container from "sourcegraph/Container";
import DefStore from "sourcegraph/def/DefStore";
import s from "sourcegraph/def/styles/Def.css";
import {qualifiedNameAndType} from "sourcegraph/def/Formatter";
import {defPath} from "sourcegraph/def";
import * as DefActions from "sourcegraph/def/DefActions";
import Dispatcher from "sourcegraph/Dispatcher";
import TimeAgo from "sourcegraph/util/TimeAgo";
import {Avatar} from "sourcegraph/components";
import RefLocationsList from "sourcegraph/def/RefLocationsList";
import {urlToDef} from "sourcegraph/def/routes";

class DefPopup extends Container {
	static propTypes = {
		def: React.PropTypes.object.isRequired,
		refLocations: React.PropTypes.array,
		path: React.PropTypes.string.isRequired,
	};

	static contextTypes = {
		features: React.PropTypes.object.isRequired,
	};

	reconcileState(state, props) {
		Object.assign(state, props);
		state.defObj = props.def;
		state.repo = props.def ? props.def.Repo : null;
		state.rev = props.def ? props.def.CommitID : null;
		state.def = props.def ? defPath(props.def) : null;

		state.authors = DefStore.authors.get(state.repo, state.rev, state.def);
	}

	onStateTransition(prevState, nextState) {
		if (prevState.repo !== nextState.repo || prevState.rev !== nextState.rev || prevState.def !== nextState.def) {
			if (this.context.features.Authors) {
				Dispatcher.Backends.dispatch(new DefActions.WantDefAuthors(nextState.repo, nextState.rev, nextState.def));
			}
		}
	}

	stores() { return [DefStore]; }

	render() {
		let def = this.props.def;
		let refLocs = this.props.refLocations;
		let authors = this.state.authors ? this.state.authors.DefAuthors || [] : null;

		return (
			<div className={s.marginBox}>
				<header className={s.boxTitle}>
					<Link to={`${urlToDef(this.state.defObj)}/-/info`}><span styleName="def-title">{qualifiedNameAndType(def, {unqualifiedNameClass: s.defName})}</span></Link>
				</header>
				<header className={s.sectionTitle}>Used in</header>

				{!refLocs && <span styleName="loading">Loading...</span>}
				{refLocs && refLocs.length === 0 && <i>No usages found</i>}
				{<RefLocationsList def={def} refLocations={refLocs} repo={this.state.repo} path={this.state.path} />}

				{authors && <header className={s.sectionTitle}>Authors</header>}
				{!authors && <span styleName="loading">Loading...</span>}
				{authors && authors.length === 0 && <i>No authors found</i>}
				{authors && authors.length > 0 &&
					<ol className={s.personList}>
						{authors.map((a, i) => (
							<li key={i} className={s.author}>
								<span className={s.badgeMinWidthWrapper}><span className={s.bytesProportion}>{Math.round(100 * a.BytesProportion)}%</span></span> <Avatar size="tiny" img={a.AvatarURL} /> {a.Email} <TimeAgo time={a.LastCommitDate} className={s.timestamp} />
							</li>
						))}
					</ol>
				}
			</div>
		);
	}
}

export default CSSModules(DefPopup, s);
